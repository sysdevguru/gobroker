package trading

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TradingTestSuite struct {
	dbtest.Suite
	asset   *models.Asset
	account *models.TradeAccount
}

func TestTradingTestSuite(t *testing.T) {
	suite.Run(t, new(TradingTestSuite))
}

func (s *TradingTestSuite) SetupSuite() {
	s.SetupDB()
	// high roller
	amt, _ := decimal.NewFromString("10000000")
	apexAcct := "apca_test"
	acc := models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
			},
		},
	}
	if err := db.DB().Create(&acc).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.account, _ = acc.ToTradeAccount()

	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}
	if err := db.DB().Create(s.asset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *TradingTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *TradingTestSuite) genOrder(orderType enum.OrderType, side enum.Side) *models.Order {
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

func (s *TradingTestSuite) genExecution(
	execType enum.ExecutionType,
	side enum.Side,
	order *models.Order) *models.Execution {

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
	}
	db.DB().Create(exec)
	return exec
}

func (s *TradingTestSuite) TestTrading() {

	// market order, single fill
	{
		order := s.genOrder(enum.Market, enum.Buy)
		exec := s.genExecution(enum.ExecutionNew, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		exec = s.genExecution(enum.ExecutionFill, enum.Buy, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		q := db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		require.Nil(s.T(), q.Error)
		require.Len(s.T(), positions, 1)
		assert.True(s.T(), exec.Qty.Equal(positions[0].Qty))
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// limit order, 4 partial fills
	{
		order := s.genOrder(enum.Limit, enum.Buy)
		for i := 0; i < 4; i++ {
			exec := s.genExecution(enum.ExecutionPartialFill, enum.Buy, order)
			err := ProcessExecution(db.DB(), s.account, exec)
			assert.Nil(s.T(), err)

			positions := []models.Position{}
			q := db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
			require.Nil(s.T(), q.Error)
			require.Len(s.T(), positions, i+1)
			assert.Equal(s.T(), order.Qty.Div(decimal.NewFromFloat(4)).String(), positions[0].Qty.String())
		}
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// fill buy order, fill sell of same position
	{
		order := s.genOrder(enum.Market, enum.Buy)
		exec := s.genExecution(enum.ExecutionFill, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		exec = s.genExecution(enum.ExecutionFill, enum.Sell, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		require.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)

		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Closed).Find(&positions)
		assert.Len(s.T(), positions, 1)
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// partial fill, expire, sell partially filled shares
	{
		order := s.genOrder(enum.Limit, enum.Buy)
		exec := s.genExecution(enum.ExecutionPartialFill, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		exec = s.genExecution(enum.ExecutionExpired, enum.Buy, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		require.Len(s.T(), positions, 1)
		assert.Equal(s.T(), decimal.NewFromFloat(25), positions[0].Qty)

		order.Qty = decimal.NewFromFloat(25)
		exec = s.genExecution(enum.ExecutionFill, enum.Sell, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Closed).Find(&positions)
		assert.Len(s.T(), positions, 1)
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// stop limit order, cancellation
	{
		order := s.genOrder(enum.StopLimit, enum.Buy)
		exec := s.genExecution(enum.ExecutionCanceled, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// partial fill, then cancel rest of order, sell the filled shares
	{
		order := s.genOrder(enum.Limit, enum.Buy)
		exec := s.genExecution(enum.ExecutionPartialFill, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		exec = s.genExecution(enum.ExecutionCanceled, enum.Buy, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		require.Len(s.T(), positions, 1)
		assert.Equal(s.T(), decimal.NewFromFloat(25), positions[0].Qty)

		order.Qty = decimal.NewFromFloat(25)
		exec = s.genExecution(enum.ExecutionFill, enum.Sell, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Closed).Find(&positions)
		assert.Len(s.T(), positions, 1)
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// buy rejection
	{
		order := s.genOrder(enum.StopLimit, enum.Buy)
		exec := s.genExecution(enum.ExecutionRejected, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)
	}

	require.Nil(s.T(), db.DB().Exec("TRUNCATE TABLE positions").Error)

	// partial fill sell, then full fill sell of remaining position
	{
		order := s.genOrder(enum.Market, enum.Buy)
		exec := s.genExecution(enum.ExecutionFill, enum.Buy, order)
		err := ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		positions := []models.Position{}
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Len(s.T(), positions, 1)

		order = s.genOrder(enum.Market, enum.Sell)
		exec = s.genExecution(enum.ExecutionPartialFill, enum.Sell, order)
		err = ProcessExecution(db.DB(), s.account, exec)
		assert.Nil(s.T(), err)

		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Closed).Find(&positions)
		assert.Len(s.T(), positions, 1)
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Len(s.T(), positions, 1)

		// 3 more partial fills to close the position
		for i := 0; i < 3; i++ {
			require.Nil(s.T(), ProcessExecution(db.DB(), s.account, s.genExecution(enum.ExecutionPartialFill, enum.Sell, order)))
		}

		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Closed).Find(&positions)
		assert.Len(s.T(), positions, 4)
		db.DB().Where("account_id = ? AND status = ?", s.account.ID, models.Open).Find(&positions)
		assert.Empty(s.T(), positions)
	}
}
