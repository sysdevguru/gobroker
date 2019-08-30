package backup

import (
	"io"
	"testing"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/valyala/fasthttp"
)

type BackupWorkerTestSuite struct {
	dbtest.Suite
	account *models.Account
	asset   *models.Asset
}

func TestBackupWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(BackupWorkerTestSuite))
}

func (s *BackupWorkerTestSuite) SetupSuite() {
	env.RegisterDefault("BROKER_MODE", "DEV")
	env.RegisterDefault("BROKER_SECRET", "79sf697d6f978yf97sh9we7gfg97wfg7")
	s.SetupDB()

	apexAcct := "apex_account"
	legalName := "First Last"
	google := "Google Inc."
	position := "CEO"
	googleAddr := "1600 Amphitheatre Parkway, Mountain View, CA"
	employed := models.Employed
	function := "runs the place"
	city := "Somewhere"
	state := "SW"
	zip := "12345"
	phone := "650-111-1111"
	ssn, err := encryption.EncryptWithKey([]byte("600-00-0001"), []byte(env.GetVar("BROKER_SECRET")))

	require.Nil(s.T(), err)

	s.account = &models.Account{
		ApexAccount: &apexAcct,
		Plan:        enum.RegularAccount,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
				Details: models.OwnerDetails{
					LegalName:        &legalName,
					PhoneNumber:      &phone,
					Employer:         &google,
					EmployerAddress:  &googleAddr,
					EmploymentStatus: &employed,
					Position:         &position,
					Function:         &function,
					StreetAddress:    address.Address([]string{"123 Somewhere Ln"}),
					City:             &city,
					State:            &state,
					PostalCode:       &zip,
					HashSSN:          &ssn,
				},
			},
		},
	}

	require.Nil(s.T(), db.DB().Create(s.account).Error)

	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}

	require.Nil(s.T(), db.DB().Create(s.asset).Error)
}

func (s *BackupWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *BackupWorkerTestSuite) TestBackupAccount() {
	asOf := time.Now()

	worker := &backupWorker{
		asOf: asOf,
		uploadS3: func(file io.ReadSeeker, path string) error {
			return nil
		},
		parallelism: 1,
	}

	worker.backupAccount(s.account)
}

func (s *BackupWorkerTestSuite) TestBackupTrades() {
	asOf := time.Now()

	order := s.genOrder(enum.Market, enum.Buy)
	exec := s.genExecution(enum.ExecutionNew, enum.Buy, order, nil)
	s.genExecution(enum.ExecutionFill, enum.Buy, order, &exec.ID)

	worker := &backupWorker{
		asOf: asOf,
		uploadS3: func(file io.ReadSeeker, path string) error {
			return nil
		},
		parallelism: 1,
	}

	worker.backupTrades(s.account)
}

func (s *BackupWorkerTestSuite) TestBackupStatementsAndConfirms() {
	asOf := time.Now()

	// success
	{
		worker := &backupWorker{
			asOf: asOf,
			uploadS3: func(file io.ReadSeeker, path string) error {
				return nil
			},
			downloadS3: func(local, remote string) error {
				return nil
			},
			uploadEgnyte: func(filePath string, data []byte) error {
				return nil
			},
			getDocuments: func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error) {
				return []apex.Document{
					apex.Document{
						URL:     "some_url",
						Account: *s.account.ApexAccount,
						Date:    asOf.Format(apexFormat),
					},
				}, nil
			},
			getHTTP: func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
				return fasthttp.StatusOK, []byte("some pdf body"), nil
			},
			parallelism: 1,
		}

		worker.backupStatements(s.account)
		worker.backupConfirms(s.account)
	}

	// upload failure
	{
		worker := &backupWorker{
			asOf: asOf,
			uploadS3: func(file io.ReadSeeker, path string) error {
				return nil
			},
			downloadS3: func(local, remote string) error {
				return nil
			},
			uploadEgnyte: func(filePath string, data []byte) error {
				return nil
			},
			getDocuments: func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error) {
				return []apex.Document{
					apex.Document{
						URL:     "some_url",
						Account: *s.account.ApexAccount,
						Date:    asOf.Format(apexFormat),
					},
				}, nil
			},
			getHTTP: func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
				return fasthttp.StatusOK, []byte("some pdf body"), nil
			},
			parallelism: 1,
		}

		worker.backupStatements(s.account)
		worker.backupConfirms(s.account)
	}

	// get docs failure
	{
		worker := &backupWorker{
			asOf: asOf,
			uploadS3: func(file io.ReadSeeker, path string) error {
				return nil
			},
			downloadS3: func(local, remote string) error {
				return nil
			},
			uploadEgnyte: func(filePath string, data []byte) error {
				return nil
			},
			getDocuments: func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error) {
				return []apex.Document{
					apex.Document{
						URL:     "some_url",
						Account: *s.account.ApexAccount,
						Date:    asOf.Format(apexFormat),
					},
				}, nil
			},
			getHTTP: func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
				return fasthttp.StatusOK, []byte("some pdf body"), nil
			},
			parallelism: 1,
		}

		worker.backupStatements(s.account)
		worker.backupConfirms(s.account)
	}

	// invalid doc date
	{
		worker := &backupWorker{
			asOf: asOf,
			uploadS3: func(file io.ReadSeeker, path string) error {
				return nil
			},
			downloadS3: func(local, remote string) error {
				return nil
			},
			uploadEgnyte: func(filePath string, data []byte) error {
				return nil
			},
			getDocuments: func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error) {
				return []apex.Document{
					apex.Document{
						URL:     "some_url",
						Account: *s.account.ApexAccount,
						Date:    asOf.Format(apexFormat),
					},
				}, nil
			},
			getHTTP: func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
				return fasthttp.StatusOK, []byte("some pdf body"), nil
			},
			parallelism: 1,
		}

		worker.backupStatements(s.account)
		worker.backupConfirms(s.account)
	}

	// http get failure
	{
		worker := &backupWorker{
			asOf: asOf,
			uploadS3: func(file io.ReadSeeker, path string) error {
				return nil
			},
			downloadS3: func(local, remote string) error {
				return nil
			},
			uploadEgnyte: func(filePath string, data []byte) error {
				return nil
			},
			getDocuments: func(account string, start, end time.Time, docType apex.DocumentType) ([]apex.Document, error) {
				return []apex.Document{
					apex.Document{
						URL:     "some_url",
						Account: *s.account.ApexAccount,
						Date:    asOf.Format(apexFormat),
					},
				}, nil
			},
			getHTTP: func(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
				return fasthttp.StatusOK, []byte("some pdf body"), nil
			},
			parallelism: 1,
		}

		worker.backupStatements(s.account)
		worker.backupConfirms(s.account)
	}
}

func (s *BackupWorkerTestSuite) genOrder(orderType enum.OrderType, side enum.Side) *models.Order {
	// store an initial order so we have something to start with
	var limitPx, stopPx *decimal.Decimal
	switch orderType {
	case enum.Stop:
		stop := decimal.NewFromFloat(99.99)
		stopPx = &stop
	case enum.Limit:
		limit := decimal.NewFromFloat(102.22)
		limitPx = &limit
	case enum.StopLimit:
		stop := decimal.NewFromFloat(99.99)
		limit := decimal.NewFromFloat(102.22)
		stopPx = &stop
		limitPx = &limit
	}
	order := &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.NewFromFloat(100),
		AssetID:     s.asset.ID,
		Symbol:      s.asset.Symbol,
		Type:        orderType,
		LimitPrice:  limitPx,
		StopPrice:   stopPx,
		Side:        side,
		TimeInForce: enum.GTC,
	}
	if err := db.DB().Create(order).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	return order
}

func (s *BackupWorkerTestSuite) genExecution(
	execType enum.ExecutionType,
	side enum.Side,
	order *models.Order,
	prevID *string) *models.Execution {

	px := decimal.NewFromFloat(float64(100))
	qty := decimal.Zero
	ordStatus := enum.OrderNew
	switch execType {
	case enum.ExecutionNew:
		ordStatus = enum.OrderNew
	case enum.ExecutionPartialFill:
		ordStatus = enum.OrderPartiallyFilled
		// partial fill a quarter of the shares
		qty = order.Qty.DivRound(decimal.NewFromFloat(float64(4)), 0)
		if order.LimitPrice != nil {
			px = *order.LimitPrice
		}
	case enum.ExecutionFill:
		ordStatus = enum.OrderFilled
		qty = order.Qty
		if order.LimitPrice != nil {
			px = *order.LimitPrice
		}
	case enum.ExecutionCanceled:
		ordStatus = enum.OrderCanceled
	case enum.ExecutionReplaced:
		ordStatus = enum.OrderReplaced
	case enum.ExecutionStopped:
		ordStatus = enum.OrderStopped
	case enum.ExecutionRejected:
		ordStatus = enum.OrderRejected
	case enum.ExecutionSuspended:
		ordStatus = enum.OrderSuspended
	case enum.ExecutionPendingNew:
		ordStatus = enum.OrderPendingNew
	case enum.ExecutionCalculated:
		ordStatus = enum.OrderCalculated
	case enum.ExecutionExpired:
		ordStatus = enum.OrderExpired
	case enum.ExecutionPendingReplace:
		ordStatus = enum.OrderPendingReplace
	}
	now := clock.Now()
	leaves := decimal.Zero
	cum := qty
	if order.FilledQty != nil {
		leaves = order.Qty.Sub(*order.FilledQty).Sub(qty)
		cum = cum.Add(*order.FilledQty)
	}
	exec := &models.Execution{
		Account:         order.Account,
		Type:            execType,
		Symbol:          "AAPL",
		Side:            side,
		OrderID:         order.ID,
		OrderType:       order.Type,
		OrderStatus:     ordStatus,
		Price:           &px,
		Qty:             &qty,
		AvgPrice:        &px,
		LeavesQty:       &leaves,
		CumQty:          &cum,
		TransactionTime: now,
		PreviousExecID:  prevID,
	}
	db.DB().Create(exec)
	return exec
}
