package rmq

import (
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/core"
	"github.com/streadway/amqp"
)

// Consume starts a blocking routine consuming messages from the specified
// queue, and processing them using the specified consumption function.
// Consume also starts a separate goroutine to reconnect if the connection
// drops, and never
func Consume(consumerName, queueName string, consumeFunc func(msg []byte) error) {
	conn := core.Connect()
	errC := make(chan *amqp.Error, 1)
	conn.NotifyClose(errC)

	// this select statement will block until an error condition is
	// reached. once this happens, the connection will be closed, and
	// we will atempt to establish a new connection.
	select {
	case err := <-consume(conn, errC, consumerName, queueName, consumeFunc):
		log.Warn(
			"rabbitmq consumption failure",
			"error", err)
		break
	case err := <-errC:
		if err != nil {
			log.Warn(
				"unexpected rabbitmq closure",
				"error", err.Error(),
			)
		} else {
			log.Info("graceful rabbitmq closure")
		}
		break
	}

	// close the dead conn, a new one will be opened
	conn.Close()
	Consume(consumerName, queueName, consumeFunc)
}

func consume(
	conn *amqp.Connection,
	mqErrC chan *amqp.Error,
	consumerName,
	queueName string,
	consumeFunc func(msg []byte) error) (errC chan error) {

	errC = make(chan error, 1)

	c, err := conn.Channel()
	if err != nil {
		errC <- err
		return
	}

	c.NotifyClose(mqErrC)
	defer c.Close()

	q, err := c.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		errC <- err
		return
	}
	if err = c.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		errC <- err
		return
	}

	msgC, err := c.Consume(
		q.Name,       // queue
		consumerName, // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		errC <- err
		return
	}
	for msg := range msgC {
		if err = consumeFunc(msg.Body); err == nil {
			if err := msg.Ack(false); err != nil {
				errC <- err
				return
			}
		} else {
			errC <- err
			return
		}
	}
	return
}
