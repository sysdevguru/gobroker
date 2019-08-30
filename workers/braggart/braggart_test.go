package braggart

import (
	"fmt"
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BraggartWorkerTestSuite struct {
	dbtest.Suite
	asset *models.Asset
}

func TestBraggartWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(BraggartWorkerTestSuite))
}

func (s *BraggartWorkerTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *BraggartWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *BraggartWorkerTestSuite) TestBraggartWorker() {
	acctSrv := account.Service().WithTx(db.DB())

	acct, err := acctSrv.Create(
		"test@example.com",
		uuid.Must(uuid.NewV4()),
	)
	require.Nil(s.T(), err)

	apexAcct := "apex_acct"
	acct.ApexAccount = &apexAcct
	require.Nil(s.T(), db.DB().Save(acct).Error)

	qty := decimal.NewFromFloat(10)
	price := decimal.NewFromFloat(22.22)
	symbol := "AAPL"
	now := clock.Now()

	executions := []*models.Execution{
		&models.Execution{
			Account:         *acct.ApexAccount,
			Side:            enum.Sell,
			Qty:             &qty,
			Price:           &price,
			TransactionTime: now,
			Symbol:          symbol,
			OrderID:         uuid.Must(uuid.NewV4()).String(),
			BrokerExecID:    uuid.Must(uuid.NewV4()).String(),
			Type:            enum.ExecutionPartialFill,
			BrokerOrderID:   uuid.Must(uuid.NewV4()).String(),
			OrderStatus:     enum.OrderPartiallyFilled,
		},
		&models.Execution{
			Account:         *acct.ApexAccount,
			Side:            enum.Buy,
			Qty:             &qty,
			Price:           &price,
			TransactionTime: now,
			Symbol:          symbol,
			OrderID:         uuid.Must(uuid.NewV4()).String(),
			BrokerExecID:    uuid.Must(uuid.NewV4()).String(),
			Type:            enum.ExecutionFill,
			BrokerOrderID:   uuid.Must(uuid.NewV4()).String(),
			OrderStatus:     enum.OrderFilled,
		},
	}

	for _, e := range executions {
		require.Nil(s.T(), db.DB().Create(e).Error)
	}

	worker = &braggartWorker{
		apexPostTx: func(tx []apex.BraggartTransaction) (*apex.PostTransactionsResponse, error) {
			id_0 := uuid.Must(uuid.NewV4()).String()
			id_1 := uuid.Must(uuid.NewV4()).String()
			ts := clock.Now().Format("2006-01-02T15:04:05.000")
			status := "PENDING"

			return &apex.PostTransactionsResponse{
				apex.PostTransactionsReceipt{
					ID:         &id_0,
					Timestamp:  &ts,
					Status:     &status,
					ExternalID: &executions[0].ID,
				},
				apex.PostTransactionsReceipt{
					ID:         &id_1,
					Timestamp:  &ts,
					Status:     &status,
					ExternalID: &executions[1].ID,
				},
			}, nil
		},
		apexListTx: func(q apex.BraggartTransactionQuery) (*apex.ListTransactionsResponse, error) {
			two := 2
			id_0 := uuid.Must(uuid.NewV4()).String()
			id_1 := uuid.Must(uuid.NewV4()).String()
			ts := clock.Now().Format("2006-01-02T15:04:05.000")
			status := "POSTED"

			return &apex.ListTransactionsResponse{
				Total: &two,
				Data: []apex.ListTransactionsResponseData{
					apex.ListTransactionsResponseData{
						ID:         &id_0,
						Timestamp:  &ts,
						Status:     &status,
						ExternalID: &executions[0].ID,
					},
					apex.ListTransactionsResponseData{
						ID:         &id_1,
						Timestamp:  &ts,
						Status:     &status,
						ExternalID: &executions[1].ID,
					},
				},
			}, nil
		},
	}

	for _, execution := range executions {
		fmt.Println(execution.BrokerExecID, execution.TransactionTime)
	}

	execs := make([]models.Execution, len(executions))

	for i, exec := range executions {
		execs[i] = *exec
	}

	worker.postExecutions(execs)

	postedExecs := []models.Execution{}
	db.DB().Where("braggart_id IS NOT NULL").Find(&postedExecs)
	assert.Len(s.T(), postedExecs, 2)
}
