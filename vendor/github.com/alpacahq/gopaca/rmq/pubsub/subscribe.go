package pubsub

import (
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"golang.org/x/net/context"
)

// subscribe consumes deliveries from an exclusive queue from a
// fanout exchange and sends to the application specific messages chan.
func subscribe(queue, exchange string, sessions chan *session, messages chan<- Message) {

	for sub := range sessions {
		if _, err := sub.QueueDeclare(queue, false, true, true, false, nil); err != nil {
			log.Error(
				"rmq cannot consume from exclusive queue",
				"queue", queue,
				"error", err)
			sub.Close()
			continue
		}

		routingKey := "ignored due to funout exchange"
		if err := sub.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			log.Error(
				"rmq cannot consume without a binding to exchange",
				"exchange", exchange,
				"error", err)
			sub.Close()
			continue
		}

		deliveries, err := sub.Consume(queue, "", false, true, false, false, nil)
		if err != nil {
			log.Error(
				"rmq cannot consume from queue",
				"queue", queue,
				"error", err)
			sub.Close()
			continue
		}

		for msg := range deliveries {
			messages <- msg.Body
			sub.Ack(msg.DeliveryTag, false)
		}

		sub.Close()
	}
}

// Subscribe returns a read-only channel, and a cancellation
// function to shutdown the subscribing connection.
func (ps *PubSub) Subscribe() (<-chan Message, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	msgs := make(chan Message)

	go func() {
		subscribe(ps.identity(), ps.exchange, ps.redial(ctx, env.GetVar("RMQ_URL")), msgs)
	}()

	return msgs, cancel
}
