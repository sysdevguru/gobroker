package pubsub

// publish publishes messages to a reconnecting session to a fanout exchange.
import (
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

// It receives from the application specific source of messages.
func publish(exchange string, sessions chan *session, messages <-chan Message) {
	for pub := range sessions {
		var (
			running bool
			reading = messages
			pending = make(chan Message, 1)
			confirm = make(chan amqp.Confirmation, 1)
		)

		// publisher confirms for this channel/connection
		if err := pub.Confirm(false); err != nil {
			log.Error("rmq publisher confirms not supported", "error", err)
			close(confirm) // confirms not supported, simulate by always nacking
		} else {
			pub.NotifyPublish(confirm)
		}

	Publish:
		for {
			var body Message
			select {
			case confirmed, ok := <-confirm:
				if !ok {
					break Publish
				}
				if !confirmed.Ack {
					log.Warn("nack message %d, body: %q", confirmed.DeliveryTag, string(body))
				}
				reading = messages

			case body = <-pending:
				err := pub.Publish(
					exchange,
					"",
					false,
					false,
					amqp.Publishing{
						Body: body,
					},
				)
				// Retry failed delivery on the next session
				if err != nil {
					pending <- body
					pub.Close()
					break Publish
				}

			case body, running = <-reading:
				// all messages consumed
				if !running {
					return
				}
				// work on pending delivery until ack'd
				pending <- body
				reading = nil
			}
		}

		pub.Close()
	}
}

// Publish returns a write-only channel, and a cancellation
// function to shutdown the publishing connection.
func (ps *PubSub) Publish() (chan<- Message, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	msgs := make(chan Message)

	go func() {
		publish(ps.exchange, ps.redial(ctx, env.GetVar("RMQ_URL")), msgs)
	}()

	return msgs, cancel
}
