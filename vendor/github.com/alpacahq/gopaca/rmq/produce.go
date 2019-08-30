package rmq

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/core"
	"github.com/streadway/amqp"
)

type prodCache struct {
	sync.Mutex
	conn   *amqp.Connection
	c      *amqp.Channel
	closed atomic.Value
}

func (p *prodCache) openChannel() {
	var err error
	p.conn = core.Connect()

	errC := make(chan *amqp.Error, 1)
	p.conn.NotifyClose(errC)

	p.c, err = p.conn.Channel()
	if err != nil {
		log.Fatal("could not establish a channel with rabbitMQ", "error", err)
	}

	go func() {
		// if received close notification, then also mark conn as closed to be
		// gracefully re-opened next time.
		<-errC
		p.Lock()
		defer p.Unlock()
		p.closed.Store(true)
	}()

	pc.closed.Store(false)
}

func (p *prodCache) close() {
	p.conn.Close()
	pc.closed.Store(true)
}

func (p *prodCache) isClosed() bool {
	return pc.closed.Load().(bool)
}

func buildProdCache() *prodCache {
	p := &prodCache{
		closed: atomic.Value{},
	}
	p.closed.Store(true)
	return p
}

var pc *prodCache
var once sync.Once

// Produce sends a message to the specified queue, while
// caching the rabbitMQ connection. It returns an error if
// this operation fails. We handle the error cases as well as
// concurrent produce requests. produce() acquires a lock
// on the pub cache, and produces a nil error on its
// return channel if it is successful. If the operation
// fails, an error will be produced on the channel. If during
// the operation, the connection or channel is disconnected,
// the pub cache error channel will receive an update, and
// the operation will fail. If a lock cannot be acquired
// after 1 second, the operation will timeout with an error.
func Produce(queueName string, msg []byte) error {
	// we have a single production cache to re-use the same
	// connection and channel
	once.Do(func() {
		pc = buildProdCache()
		pc.openChannel()
	})

	// we acquire a lock on the pub cache in case an error
	// occurs and the connection needs to be
	pc.Lock()
	defer pc.Unlock()

	select {
	case err := <-produce(queueName, msg):
		if err != nil {
			log.Warn(
				"rabbitmq produce failure",
				"error", err)
		}
		return err
	case <-time.After(time.Second):
		return errors.New("rmq produce timeout")
	}
}

// non-blocking call to produce a message to a queue
func produce(queueName string, msg []byte) (errC chan error) {
	errC = make(chan error, 1)

	if pc.isClosed() {
		pc.openChannel()
	}

	q, err := pc.c.QueueDeclare(
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

	errC <- pc.c.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
		})
	return
}
