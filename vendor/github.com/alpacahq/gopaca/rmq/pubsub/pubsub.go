package pubsub

import (
	"context"
	"fmt"
	"sync"

	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/core"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

var (
	once sync.Once
	id   string
)

// Message is a message passed via rabbitMQ
type Message []byte

// session composes an amqp.Connection with an amqp.Channel
type session struct {
	*amqp.Connection
	*amqp.Channel
	done chan interface{}
}

// Close tears the connection down, taking the channel with it.
func (s *session) Close() error {
	var err error
	if s.Connection != nil {
		err = s.Connection.Close()
	}
	s.done <- struct{}{}
	return err
}

type PubSub struct {
	exchange string
}

func NewPubSub(exchange string) *PubSub {
	return &PubSub{exchange: exchange}
}

// identity returns the same host/process unique string for the lifetime of
// this process so that subscriber reconnections reuse the same queue name.
func (ps *PubSub) identity() string {
	once.Do(func() {
		id = uuid.Must(uuid.NewV4()).String()
	})

	return fmt.Sprintf("%s-%s", ps.exchange, id)
}

// redial continually connects to the URL, exiting the program when no longer possible
func (ps *PubSub) redial(ctx context.Context, url string) chan *session {
	sessions := make(chan *session)

	go func() {
		defer close(sessions)

		for {
			newSession, err := createSession(ps.exchange)
			if err != nil {
				log.Warn(
					"rmq failed to create session",
					"error", err,
				)
				continue
			}

			sessions <- newSession

			select {
			case <-newSession.done:
				// go to next loop to grab new session
			case <-ctx.Done():
				newSession.Close()
				return
			}
		}
	}()

	return sessions
}

func createSession(exchange string) (*session, error) {
	conn := core.Connect()

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create channel")
	}

	if err := ch.ExchangeDeclare(exchange, "fanout", false, true, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "failed to declare exchange")
	}

	return &session{Connection: conn, Channel: ch, done: make(chan interface{}, 1)}, nil
}
