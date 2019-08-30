package trade

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alpacahq/gobroker/stream"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	api "github.com/alpacahq/gobroker/rest/api/controller/order"
	"github.com/alpacahq/gobroker/service/registry"
	"github.com/alpacahq/gobroker/trading"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq"
	"github.com/alpacahq/gopaca/rmq/pubsub"
	"github.com/gofrs/uuid"
)

var (
	TestOrderCount int
)

type TradeWorker struct {
	stream                        chan<- pubsub.Message
	cancel                        context.CancelFunc
	processExecution              func(tx *gorm.DB, acct *models.TradeAccount, exec *models.Execution) error
	consume                       func(consumerName, queueName string, consumeFunc func(msg []byte) error)
	services                      registry.Registry
	queueExecutions               string
	queueCancelRejection          string
	consumerName                  string
	sendOrderExecutedNotification func(*gorm.DB, uuid.UUID, *models.Execution) error
	db                            *gorm.DB
}

// NewTradeWorker returns fix msg processing worker which is used in both gobroker and papertrader. Need to be careful
// not to mix them up.
func NewTradeWorker(
	db *gorm.DB,
	queueExecution, queueCancelRejection, queueStream, consumerName string,
	services registry.Registry,
	sendOrderExecutedNotification func(*gorm.DB, uuid.UUID, *models.Execution) error) *TradeWorker {
	worker := TradeWorker{
		queueCancelRejection:          queueCancelRejection,
		queueExecutions:               queueExecution,
		consumerName:                  consumerName,
		db:                            db,
		processExecution:              trading.ProcessExecution,
		consume:                       rmq.Consume,
		services:                      services,
		sendOrderExecutedNotification: sendOrderExecutedNotification,
	}
	worker.stream, worker.cancel = pubsub.NewPubSub(queueStream).Publish()

	go worker.workExecutions()
	go worker.workCancelRejections()

	return &worker
}

// Stop disconnects the RMQ connection and prepares the routine
// for graceful shutdown
func (w *TradeWorker) Stop() {
	w.cancel()
}

// this processes incoming executions, which need to be stored to the DB,
// posted to Braggart, and then the order that the execution corresponds
// to needs to have its status updated.
func (w *TradeWorker) workExecutions() {
	handler := func(msg []byte) error {
		e := &models.Execution{}
		if err := json.Unmarshal(msg, e); err != nil {
			// we couldn't even unmarshal it, so let's store the raw
			// message alone since we don't have order or acct ID
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueExecutions,
				Body:   msg,
				Reason: models.MarshalFailure,
				Error:  err.Error(),
			})
			return err
		}

		if e.ID == "TEST ORDER" {
			TestOrderCount++
			v := models.Validation{
				Count: 0,
				GBID:  e.OrderID,
			}
			data, err := json.Marshal(v)
			if err != nil {
				return err
			}
			log.Info("Received TEST TRADE confirm", "GBID:", e.OrderID)
			rmq.Produce("VALIDATION", data)
			return nil
		}

		tx := w.db.Begin()

		q := tx.Where(
			"broker_exec_id = ? AND transaction_time = ?",
			e.BrokerExecID, e.TransactionTime).Find(e)

		if !q.RecordNotFound() {
			// already handled
			tx.Rollback()
			return nil
		}

		if q.Error != nil && !gorm.IsRecordNotFoundError(q.Error) {
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:   w.queueExecutions,
				Body:    msg,
				Reason:  models.DatabaseFailure,
				Error:   fmt.Sprintf("failed to query broker_exec_id: %v (%v)", e.BrokerExecID, q.Error),
				OrderID: &e.OrderID,
			})
			return q.Error
		}

		aSrv := w.services.Account().WithTx(tx).ForUpdate()

		acct, err := aSrv.GetByApexAccount(e.Account)
		if err != nil {
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:   w.queueExecutions,
				Body:    msg,
				Reason:  models.DatabaseFailure,
				Error:   err.Error(),
				OrderID: &e.OrderID,
			})
			return err
		}

		if err := tx.Create(e).Error; err != nil {
			// we couldn't store it, so let's store the raw
			// message alone since we don't have order or acct ID
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:   w.queueExecutions,
				Body:    msg,
				Reason:  models.DatabaseFailure,
				Error:   err.Error(),
				OrderID: &e.OrderID,
			})
			return err
		}

		log.Debug(
			"trade worker new execution",
			"account", acct.ID,
			"type", e.Type,
			"order", e.OrderID)

		update, err := w.handleExecution(tx, acct, e)

		if err != nil {
			// we failed to handle the execution, so we should store
			// it to the failure table and report
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:     w.queueExecutions,
				Body:      msg,
				Reason:    models.RMQFailure,
				Error:     err.Error(),
				AccountID: &acct.ID,
				OrderID:   &e.OrderID,
			})
			return err
		}

		if err := w.streamPush(stream.OutboundMessage{
			Stream: stream.TradeUpdatesStream(acct.IDAsUUID()),
			Data:   update,
		}); err != nil {
			tx.Rollback()
			return err
		}

		//email should be sent here
		if e.Type == enum.ExecutionFill {
			if err := w.sendOrderExecutedNotification(tx, acct.IDAsUUID(), e); err != nil {
				tx.Rollback()
				return err
			}
		}

		return tx.Commit().Error
	}
	w.consume(w.consumerName, w.queueExecutions, handler)
}

// this processes incoming cancel rejections. we need to notify the
// user via stream, but no other action is required
func (w *TradeWorker) workCancelRejections() {
	handler := func(msg []byte) error {
		m := map[string]interface{}{}

		if err := json.Unmarshal(msg, &m); err != nil {
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.MarshalFailure,
				Error:  err.Error(),
			})
			return err
		}

		orderID, err := uuid.FromString(m["order_id"].(string))
		if err != nil {
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.MarshalFailure,
				Error:  err.Error(),
			})
		}

		tx := w.db.Begin()

		order := &models.Order{}

		q := tx.Where("id = ?", orderID.String()).Find(order)

		if q.RecordNotFound() {
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.DatabaseFailure,
				Error:  err.Error(),
			})
			return err
		}

		if q.Error != nil {
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.DatabaseFailure,
				Error:  fmt.Sprintf("failed to query order (%s)", err.Error()),
			})
			return q.Error
		}

		aSrv := w.services.Account().WithTx(tx).ForUpdate()

		acct, err := aSrv.GetByApexAccount(order.Account)
		if err != nil {
			tx.Rollback()
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.DatabaseFailure,
				Error:  err.Error(),
			})
			return err
		}

		// When canceled order which not sent to fix gateway for some reason.
		// We'll mark them canceled, and let user know it is canceled.
		if order.Status == enum.OrderAccepted {
			canceledAt := clock.Now()

			patch := models.Order{
				Status:     enum.OrderCanceled,
				CanceledAt: &canceledAt,
			}

			if err := tx.Model(order).Updates(patch).Error; err != nil {
				tx.Rollback()
				w.storeFailure(&models.TradeFailure{
					Queue:  w.queueCancelRejection,
					Body:   msg,
					Reason: models.DatabaseFailure,
					Error:  err.Error(),
				})
				return errors.Wrap(err, "failed to update order status to canceled")
			}

			report := map[string]interface{}{
				"event":     enum.ExecutionCanceled,
				"order":     api.OrderToEntity(order, w.services.AssetCache().Get(order.AssetID)),
				"timestamp": order.CanceledAt,
			}

			if err := w.streamPush(stream.OutboundMessage{
				Stream: stream.TradeUpdatesStream(acct.IDAsUUID()),
				Data:   report,
			}); err != nil {
				tx.Rollback()
				return err
			}

			if err := tx.Commit().Error; err != nil {
				w.storeFailure(&models.TradeFailure{
					Queue:  w.queueCancelRejection,
					Body:   msg,
					Reason: models.DatabaseFailure,
					Error:  err.Error(),
				})
				return errors.Wrap(err, "failed to commit order status to canceled")
			}

			return nil
		}

		if err = tx.Commit().Error; err != nil {
			w.storeFailure(&models.TradeFailure{
				Queue:  w.queueCancelRejection,
				Body:   msg,
				Reason: models.DatabaseFailure,
				Error:  err.Error(),
			})
			return err
		}

		log.Debug("trade worker new cancel rejection", "account", acct.ID, "order", order.ID)

		return w.streamPush(stream.OutboundMessage{
			Stream: stream.TradeUpdatesStream(acct.IDAsUUID()),
			Data: map[string]interface{}{
				"event":  "order_cancel_rejected",
				"reason": m["reason"],
				"order":  api.OrderToEntity(order, w.services.AssetCache().Get(order.AssetID)),
			},
		})
	}
	w.consume(w.consumerName, w.queueCancelRejection, handler)
}

func (w *TradeWorker) handleExecution(tx *gorm.DB, acct *models.TradeAccount, e *models.Execution) (map[string]interface{}, error) {
	srv := w.services.Order().WithTx(tx)

	order, err := srv.GetByID(acct.IDAsUUID(), e.OrderIDAsUUID())
	if err != nil {
		log.Error(
			"trade worker failed to find order for execution",
			"order", e.OrderID,
			"error", err)
		return nil, err
	}

	if err := tx.Save(order.Update(e)).Error; err != nil {
		return nil, err
	}

	if err := w.processExecution(tx, acct, e); err != nil {
		log.Error(
			"trade worker process failure",
			"order", order.ID,
			"account", acct.ID,
			"error", err)
		return nil, err
	}

	report := map[string]interface{}{
		"event": e.Type,
		"order": api.OrderToEntity(order, w.services.AssetCache().Get(order.AssetID)),
	}

	switch e.Type {
	case enum.ExecutionPartialFill:
		fallthrough
	case enum.ExecutionFill:
		report["timestamp"] = *order.FilledAt
		report["price"] = *order.FilledAvgPrice
	case enum.ExecutionCanceled:
		report["timestamp"] = *order.CanceledAt
	case enum.ExecutionExpired:
		fallthrough
	case enum.ExecutionRejected:
		report["timestamp"] = *order.FailedAt
	}
	return report, nil
}

func (w *TradeWorker) streamPush(msg stream.OutboundMessage) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	w.stream <- pubsub.Message(buf)

	return nil
}

func (w *TradeWorker) storeFailure(failure *models.TradeFailure) {
	tx := w.db.Begin().Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;")

	for {
		if err := tx.Create(failure).Error; err == nil {
			break
		} else {
			tx.Rollback()
			if db.IsSerializabilityError(err) {
				continue
			}
			log.Error("failed to store failed execution", "error", err, "report", *failure)
			break
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error("failed to store failed execution", "error", err, "report", *failure)
	}
}
