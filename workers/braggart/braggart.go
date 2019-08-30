package braggart

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/workers/common"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type braggartWorker struct {
	apexPostTx func(tx []apex.BraggartTransaction) (*apex.PostTransactionsResponse, error)
	apexListTx func(q apex.BraggartTransactionQuery) (*apex.ListTransactionsResponse, error)
	done       chan struct{}
}

var (
	worker           *braggartWorker
	postableStatuses = []enum.ExecutionType{
		enum.ExecutionFill,
		enum.ExecutionPartialFill,
	}
)

func Work() {
	if worker == nil {
		worker = &braggartWorker{
			apexPostTx: apex.Client().PostTransactions,
			apexListTx: apex.Client().ListTransactions,
			done:       make(chan struct{}, 1),
		}
		worker.done <- struct{}{}
	}

	// make sure not to overlap if the work routine is taking long
	if common.WaitTimeout(worker.done, time.Second) {
		// timed out, so let's skip this round and wait until it finishes
		return
	}

	defer func() {
		worker.done <- struct{}{}
	}()

	if !utils.Dev() {
		executions := []models.Execution{}

		q := db.DB().Where(
			"type IN (?) AND qty > ? AND braggart_timestamp IS NULL",
			postableStatuses,
			decimal.Zero).Find(&executions).Order(
			"created_at DESC").Limit(1000)

		if q.Error == nil && len(executions) > 0 {
			log.Info("posting executions to braggart", "count", len(executions))
			worker.postExecutions(executions)
			log.Info("braggart executions posted")
		}
	}
}

// PostExecutions is a public wrapper for postExecutions
// to be used in integration testing.
func PostExecutions(executions []models.Execution) {
	if worker == nil {
		worker = &braggartWorker{
			apexPostTx: apex.Client().PostTransactions,
			apexListTx: apex.Client().ListTransactions,
		}
	}
	worker.postExecutions(executions)
}

func (w *braggartWorker) postExecutions(executions []models.Execution) {
	bragBatch := make([]apex.BraggartTransaction, len(executions))
	for i, e := range executions {
		tx, err := e.Braggart()
		if err != nil {
			log.Error("braggart worker transaction generation failure", "error", err)
			continue
		}
		bragBatch[i] = *tx
	}
	if resp, err := w.apexPostTx(bragBatch); err != nil {
		log.Error("braggart worker post failure", "count", len(bragBatch), "error", err)
	} else {
		for _, receipt := range *resp {
			if err := w.handleBraggart(executions, receipt); err != nil {
				log.Error("braggart worker post response handle failure", "error", err)
			}
		}
	}
}

func (w *braggartWorker) handleBraggart(executions []models.Execution, receipt apex.PostTransactionsReceipt) error {
	var e models.Execution

	found := false

	for _, e = range executions {
		if e.ID == *receipt.ExternalID {
			found = true
			break
		}
	}

	if found {
		if *receipt.Status == "ERROR" {
			if strings.Contains(*receipt.ErrorDetails, "transaction already exists") {
				// already posted, skip it
				return nil
			}

			return fmt.Errorf("braggart error - %v", *receipt.ErrorDetails)
		} else {
			bragTimestamp, err := time.Parse("2006-01-02T15:04:05.000", *receipt.Timestamp)
			if err != nil {
				return err
			}
			e.BraggartID = receipt.ID
			e.BraggartStatus = receipt.Status
			e.BraggartTimestamp = &bragTimestamp
		}
		for {
			tx := db.RepeatableRead()

			if err := tx.Save(&e).Error; err != nil {
				tx.Rollback()

				if db.IsSerializabilityError(err) {
					continue
				}
				return err
			}

			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				log.Panic("failed to commit braggart update", "error", err)
			}

			break
		}
	}
	return nil
}
