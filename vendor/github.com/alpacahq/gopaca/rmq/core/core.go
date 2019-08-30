package core

import (
	"time"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/streadway/amqp"
)

// AMQP docs state that it is recommended to maintain separate connections
// for produce and consume so not to have TCP pushback on producing
// affect the ability to consume messages. As a result, here we have
// separate conns, but they are re-used for their respective tasks on
// different queues. While Consume() is meant to be called once and
// continue to run throughout the life of the program, since produce
// is called on an as-needed basis, we also cache the channel object
// generated for the connection so as to not introduce too much overhead
// on each call for the channel allocation.

var retryDuration = 5 * time.Second

// infinitely tries to reconnect to rmq using retryDuration
func Connect() (conn *amqp.Connection) {
	var err error
	retryCount := 0
	for {
		if conn, err = amqp.Dial(env.GetVar("RMQ_URL")); err == nil {
			break
		} else {
			log.Warn(
				"rabbitmq connection failure",
				"error", err,
				"retry", retryDuration.String())
			retryCount++
			<-time.After(retryDuration)
		}
	}
	if retryCount > 0 {
		log.Info("rabbitmq recovered from connection failure")
	}
	return conn
}
