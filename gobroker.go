package main

import (
	stdContext "context"
	"encoding/json"
	"flag"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/metrics/server"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest"
	"github.com/alpacahq/gobroker/service/order"
	"github.com/alpacahq/gobroker/stream"
	"github.com/alpacahq/gobroker/utils/gbevents"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/signalman"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq"
	"github.com/alpacahq/gopaca/rmq/pubsub"
	"go.uber.org/zap/zapcore"
)

func shutdown() error {
	timeout := time.Second
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
	defer cancel()
	return rest.Shutdown(ctx)
}

func init() {
	// set the clock
	clock.Set()

	rand.Seed(clock.Now().UTC().UnixNano())

	// register env defaults
	initializer.Initialize()

	flag.Parse()

	// log errors to slack
	log.Logger().AddCallback(
		"gobroker_slack_errors",
		zapcore.ErrorLevel,
		func(i interface{}) {
			msg := slack.NewServerError()
			msg.SetBody(i)
			slack.Notify(msg)
		},
	)

	// set deployment level on logger
	log.Logger().SetDeploymentLevel(env.GetVar("BROKER_MODE"))

	gbevents.RegisterSignalHandler()

	signalman.RegisterFunc("rest_shutdown", shutdown)
}

func main() {

	go func() {
		if err := server.Serve(); err != nil && err != http.ErrServerClosed {
			log.Error("stopped metrics server", "error", err)
		}
	}()

	c, cancel := pubsub.NewPubSub("stream").Subscribe()

	stream.Initialize(gbreg.Services.AccessKey(), c, cancel)

	// Send test trade and wait for it to increment the counter
	if err := sendTestTrade(); err != nil {
		panic(err)
	}

	log.Info("Waiting for test trade response for...", "GBID", models.GBID)

	go rmq.Consume("gobroker", "VALIDATION", handleTestTrade)

	for {
		if models.TestCount != 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Info("gobroker is live", "mode", env.GetVar("BROKER_MODE"), "clock", clock.Now())

	signalman.Start()

	go func() {
		gbevents.RunForever()
	}()

	if err := rest.Start(env.GetVar("BROKER_PORT"), gbreg.Services); err != nil {
		if !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("rest server unexpectedly exited", "error", err)
		}
	}

	defer db.DB().Close()

	log.Info("waiting for graceful shutdown")
	signalman.Wait()
}

func sendTestTrade() error {
	OrderRequests := env.GetVar("ORDER_REQUESTS_QUEUE")
	req := order.OrderRequest{
		RequestType: order.REQ_NEW,
		Order: &models.Order{
			ID:            "TEST ORDER",
			ClientOrderID: models.GBID,
		},
	}
	buf, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return rmq.Produce(OrderRequests, buf)
}
func handleTestTrade(msg []byte) error {
	v := models.Validation{}
	err := json.Unmarshal(msg, &v)
	if err != nil {
		return err
	}
	if v.GBID == models.GBID {
		models.TestCount++
	} else {
		v.Count++
		if v.Count > 9 {
			log.Info("failure to validate: ten retries of validation exceeded for ", "GBID", v.GBID)
			return nil
		}
		msg, err := json.Marshal(&v)
		if err != nil {
			return err
		}
		rmq.Produce("VALIDATION", msg)
	}
	return nil
}
