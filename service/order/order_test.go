package order

import (
	"math/big"
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/position"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OrderTestSuite struct {
	dbtest.Suite
	asset          *models.Asset
	order          *models.Order
	oldOrder       *models.Order
	account        *models.Account
	orderRequester func(accountID uuid.UUID, msg interface{}) error
}

func TestOrderTestSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

func (s *OrderTestSuite) SetupSuite() {
	s.orderRequester = func(accountID uuid.UUID, msg interface{}) error {
		return nil
	}

	s.SetupDB()
	amt, _ := decimal.NewFromString("1000000")
	apexAcct := "apca_test"
	name := "Test Trader"
	s.account = &models.Account{
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

	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	details := &models.OwnerDetails{
		OwnerID:   s.account.Owners[0].ID,
		LegalName: &name,
	}
	if err := db.DB().Create(details).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
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
	limitPx := decimal.NewFromFloat(float64(102.22))
	s.order = &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.NewFromFloat(float64(100)),
		AssetID:     s.asset.ID,
		Symbol:      s.asset.Symbol,
		Type:        enum.Limit,
		LimitPrice:  &limitPx,
		Side:        enum.Buy,
		TimeInForce: enum.GTC,
		SubmittedAt: clock.Now(),
	}
	s.oldOrder = &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.NewFromFloat(float64(100)),
		AssetID:     s.asset.ID,
		Symbol:      "OLD",
		Type:        enum.Limit,
		LimitPrice:  &limitPx,
		Side:        enum.Buy,
		TimeInForce: enum.GTC,
		SubmittedAt: clock.Now().AddDate(0, -1, 0),
	}
	if err := db.DB().Create(s.order).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err := db.DB().Create(s.oldOrder).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *OrderTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *OrderTestSuite) TestList() {
	srv := Service(s.orderRequester, position.Service(assetcache.GetAssetCache()), tradeaccount.Service()).WithTx(db.DB())

	status := enum.OrderClosed
	orders, err := srv.List(s.account.IDAsUUID(), status, nil, nil, nil, true)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), orders)
	assert.Empty(s.T(), orders)

	status = enum.OrderOpen
	orders, err = srv.List(s.account.IDAsUUID(), status, nil, nil, nil, false)
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), orders)
	assert.Equal(s.T(), *s.account.ApexAccount, orders[0].Account)
	for i := 1; i < len(orders); i++ {
		// should be descending
		assert.True(s.T(), orders[i].SubmittedAt.Before(orders[i-1].SubmittedAt))
	}

	until := clock.Now().AddDate(0, 0, -1)
	orders, err = srv.List(s.account.IDAsUUID(), status, &until, nil, nil, true)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), orders)
	assert.NotEmpty(s.T(), orders)
	for i := 0; i < len(orders); i++ {
		assert.True(s.T(), orders[i].SubmittedAt.Before(until))
	}

	after := clock.Now().AddDate(0, 0, -1)
	orders, err = srv.List(s.account.IDAsUUID(), status, nil, nil, &after, false)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), orders)
	assert.NotEmpty(s.T(), orders)
	for i := 0; i < len(orders); i++ {
		assert.True(s.T(), orders[i].SubmittedAt.After(after))
	}

	orders, err = srv.List(s.account.IDAsUUID(), status, nil, nil, nil, true)
	for i := 1; i < len(orders); i++ {
		// should be ascending
		assert.True(s.T(), orders[i].SubmittedAt.After(orders[i-1].SubmittedAt))
	}

	orders, err = srv.List(uuid.Must(uuid.NewV4()), nil, nil, nil, nil, false)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), orders)
}

func (s *OrderTestSuite) TestCreate() {
	srv := Service(s.orderRequester, position.Service(assetcache.GetAssetCache()), tradeaccount.Service()).WithTx(db.DB())

	o := &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.NewFromFloat(float64(50)),
		AssetID:     s.asset.ID,
		Symbol:      s.asset.Symbol,
		Type:        enum.Limit,
		LimitPrice:  s.order.LimitPrice,
		Side:        enum.Buy,
		TimeInForce: enum.GTC,
	}

	order, err := srv.Create(s.account.IDAsUUID(), o)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), order)
	assert.Equal(s.T(), order.Status, enum.OrderNew)
	assert.True(s.T(), o.SubmittedAt.Before(clock.Now()))

	order, err = srv.Create(uuid.Must(uuid.NewV4()), o)
	assert.Nil(s.T(), order)
	assert.NotNil(s.T(), err)

	o = &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.Zero,
		AssetID:     s.asset.ID,
		Symbol:      s.asset.Symbol,
		Type:        enum.Limit,
		LimitPrice:  s.order.LimitPrice,
		Side:        enum.Buy,
		TimeInForce: enum.GTC,
	}

	order, err = srv.Create(uuid.Must(uuid.NewV4()), o)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)

	px := o.LimitPrice.Mul(decimal.NewFromFloat(-1))
	o.LimitPrice = &px
	order, err = srv.Create(uuid.Must(uuid.NewV4()), o)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)
}

func (s *OrderTestSuite) TestLiquidationOnly() {
	accountNumber := "3AP05024"
	acct := &models.TradeAccount{
		ApexAccount: &accountNumber,
	}
	env.RegisterDefault("LIQUIDATION_ACCOUNT_NUMBERS", accountNumber)

	var res bool
	res = liquidationOnly(acct)
	assert.True(s.T(), res)

	env.RegisterDefault("LIQUIDATION_ACCOUNT_NUMBERS", "")

	res = liquidationOnly(acct)
	assert.False(s.T(), res)
}

func (s *OrderTestSuite) TestCancel() {
	srv := Service(s.orderRequester, position.Service(assetcache.GetAssetCache()), tradeaccount.Service()).WithTx(db.DB())

	o := &models.Order{
		Account:     *s.account.ApexAccount,
		Qty:         decimal.NewFromFloat(float64(25)),
		AssetID:     s.asset.ID,
		Symbol:      s.asset.Symbol,
		Type:        enum.Limit,
		LimitPrice:  s.order.LimitPrice,
		Side:        enum.Buy,
		TimeInForce: enum.GTC,
	}

	order, err := srv.Create(s.account.IDAsUUID(), o)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), order)

	err = srv.Cancel(uuid.Must(uuid.NewV4()), order.IDAsUUID())
	assert.NotNil(s.T(), err)

	err = srv.Cancel(s.account.IDAsUUID(), order.IDAsUUID())
	assert.Nil(s.T(), err)

	err = srv.Cancel(s.account.IDAsUUID(), uuid.Must(uuid.NewV4()))
	assert.NotNil(s.T(), err)
}

func (s *OrderTestSuite) TestGetByID() {
	srv := Service(s.orderRequester, position.Service(assetcache.GetAssetCache()), tradeaccount.Service()).WithTx(db.DB())

	order, err := srv.GetByID(s.account.IDAsUUID(), s.order.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), order)
	assert.Equal(s.T(), order.ID, s.order.ID)

	order, err = srv.GetByID(uuid.Must(uuid.NewV4()), s.order.IDAsUUID())
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)

	order, err = srv.GetByID(s.account.IDAsUUID(), uuid.Must(uuid.NewV4()))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)
}

func (s *OrderTestSuite) TestGetByClientOrderID() {
	srv := Service(s.orderRequester, position.Service(assetcache.GetAssetCache()), tradeaccount.Service()).WithTx(db.DB())

	order, err := srv.GetByClientOrderID(s.account.IDAsUUID(), s.order.ClientOrderID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), order)
	assert.Equal(s.T(), order.ID, s.order.ID)

	order, err = srv.GetByClientOrderID(uuid.Must(uuid.NewV4()), s.order.ClientOrderID)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)

	order, err = srv.GetByClientOrderID(s.account.IDAsUUID(), uuid.Must(uuid.NewV4()).String())
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), order)
}

func (s *OrderTestSuite) TestAvailableQty() {
	amt, _ := decimal.NewFromString("1000000")
	apexAcct := "avail_check"
	acct := &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader+testAvailableQty@example.com",
				Primary: true,
			},
		},
	}
	assert.Nil(s.T(), db.DB().Create(acct).Error)

	pos := &models.Position{
		AssetID:        s.asset.IDAsUUID(),
		AccountID:      acct.ID,
		Status:         models.Open,
		Side:           models.Long,
		Qty:            decimal.NewFromBigInt(big.NewInt(int64(100)), 0),
		EntryPrice:     decimal.NewFromFloat(100.00),
		EntryTimestamp: clock.Now(),
		EntryOrderID:   uuid.Must(uuid.NewV4()).String(),
	}
	assert.Nil(s.T(), db.DB().Create(&pos).Error)

	// Case - No orders

	{
		o := &models.Order{
			Account:     *acct.ApexAccount,
			Qty:         decimal.NewFromFloat(float64(25)),
			AssetID:     s.asset.ID,
			Symbol:      s.asset.Symbol,
			Type:        enum.Limit,
			LimitPrice:  s.order.LimitPrice,
			Side:        enum.Sell,
			TimeInForce: enum.GTC,
		}
		ta, _ := acct.ToTradeAccount()
		assert.Nil(s.T(), checkAvailableQty(db.DB(), o, ta))

		o.Qty = decimal.NewFromFloat(float64(101))
		ta, _ = acct.ToTradeAccount()
		assert.NotNil(s.T(), checkAvailableQty(db.DB(), o, ta))

	}

	// Case - open order with 50 shares
	{
		pending := &models.Order{
			Account:     *acct.ApexAccount,
			Qty:         decimal.NewFromFloat(float64(50)),
			AssetID:     s.asset.ID,
			Symbol:      s.asset.Symbol,
			Type:        enum.Limit,
			LimitPrice:  s.order.LimitPrice,
			Side:        enum.Sell,
			TimeInForce: enum.GTC,
			Status:      enum.OrderNew,
		}
		assert.Nil(s.T(), db.DB().Create(&pending).Error)

		o := &models.Order{
			Account:     *acct.ApexAccount,
			Qty:         decimal.NewFromFloat(float64(50)),
			AssetID:     s.asset.ID,
			Symbol:      s.asset.Symbol,
			Type:        enum.Limit,
			LimitPrice:  s.order.LimitPrice,
			Side:        enum.Sell,
			TimeInForce: enum.GTC,
		}
		ta, _ := acct.ToTradeAccount()
		assert.Nil(s.T(), checkAvailableQty(db.DB(), o, ta))

		o.Qty = decimal.NewFromFloat(float64(51))
		ta, _ = acct.ToTradeAccount()
		assert.NotNil(s.T(), checkAvailableQty(db.DB(), o, ta))

		// Still check availability expected to be failed when partially filled
		pending.Status = enum.OrderPartiallyFilled
		filled := decimal.RequireFromString("10.0")
		pending.FilledQty = &filled
		assert.Nil(s.T(), db.DB().Save(&pending).Error)

		o.Qty = decimal.NewFromFloat(float64(60))
		ta, _ = acct.ToTradeAccount()
		assert.Nil(s.T(), checkAvailableQty(db.DB(), o, ta))

		o.Qty = decimal.NewFromFloat(float64(61))
		ta, _ = acct.ToTradeAccount()
		assert.NotNil(s.T(), checkAvailableQty(db.DB(), o, ta))
	}
}
