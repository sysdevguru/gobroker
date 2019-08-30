package order

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gobroker/service/position"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type OrderService interface {
	List(
		accountID uuid.UUID,
		statuses []enum.OrderStatus,
		until *time.Time,
		limit *int,
		after *time.Time,
		isAscending bool) ([]models.Order, error)
	Create(accountID uuid.UUID, order *models.Order) (*models.Order, error)
	Cancel(accountID uuid.UUID, orderID uuid.UUID) error
	GetByID(accountID uuid.UUID, orderID uuid.UUID) (*models.Order, error)
	GetByClientOrderID(accountID uuid.UUID, clientOrderID string) (*models.Order, error)
	WithTx(tx *gorm.DB) OrderService
}

type orderService struct {
	tx         *gorm.DB
	submit     OrderRequester
	posService position.PositionService
	accService tradeaccount.TradeAccountService
}

type OrderRequester func(accountID uuid.UUID, msg interface{}) error

func Service(submit OrderRequester, posService position.PositionService, accService tradeaccount.TradeAccountService) OrderService {
	return &orderService{
		submit:     submit,
		posService: posService,
		accService: accService,
	}
}

func (s *orderService) WithTx(tx *gorm.DB) OrderService {
	s.tx = tx
	return s
}

type ReqType int

const (
	REQ_NEW ReqType = iota
	REQ_CANCEL
	REQ_REPLACE
)

type OrderRequest struct {
	ReplyAddr   string
	RequestType ReqType
	Order       *models.Order
}

func (s *orderService) GetByID(accountID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	acct, err := s.accService.WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return nil, err
	}

	if acct.ApexAccount == nil {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("order not found for %v", orderID))
	}

	return op.GetOrderByID(s.tx, *acct.ApexAccount, orderID)
}

func (s *orderService) GetByClientOrderID(accountID uuid.UUID, clientOrderID string) (*models.Order, error) {
	acct, err := s.accService.WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return nil, err
	}

	if acct.ApexAccount == nil {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("order not found for %v", clientOrderID))
	}

	return op.GetOrderByClientOrderID(s.tx, *acct.ApexAccount, clientOrderID)
}

// To paginate:
//   Take the last returned order, and call List
//   with `after` set to that order's submitted_at time
func (s *orderService) List(
	accountID uuid.UUID,
	statuses []enum.OrderStatus,
	until *time.Time,
	limit *int,
	after *time.Time,
	isAscending bool,
) ([]models.Order, error) {

	acct, err := s.accService.WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return nil, err
	}

	if acct.ApexAccount == nil {
		return []models.Order{}, nil
	}

	orders := []models.Order{}

	q := s.tx.Where("account = ?", acct.ApexAccount)

	if until != nil && !until.IsZero() {
		q = q.Where("submitted_at < ?", *until)
	}

	if after != nil && !after.IsZero() {
		q = q.Where("submitted_at > ?", *after)
	}

	if statuses != nil {
		q = q.Where("status IN (?)", statuses)
	}

	if limit != nil && *limit > 0 {
		q = q.Limit(*limit)
	}

	direction := "DESC"
	if isAscending {
		direction = "ASC"
	}

	q = q.Order(fmt.Sprintf("submitted_at %s, id", direction)).Find(&orders)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return orders, nil
}

func (s *orderService) Cancel(accountID uuid.UUID, orderID uuid.UUID) error {
	tx := s.tx
	now := clock.Now()

	acc, err := s.accService.WithTx(tx).ForUpdate().GetByID(accountID)
	if err != nil {
		return err
	}

	order, err := op.GetOrderByID(tx, *acc.ApexAccount, orderID)
	if err != nil {
		return err
	}

	req := OrderRequest{
		RequestType: REQ_CANCEL,
		Order:       order,
	}
	if err := s.submit(accountID, req); err != nil {
		return gberrors.InternalServerError.WithMsg("failed to cancel order")
	}
	order.CancelRequestedAt = &now
	return tx.Save(order).Error
}

// Returns true if the account is restricted only for liquidation
// apex_account is in the list, or if no list of accounts is set,
// then always returns true
// This is a temp/quick workaround and it should be implemented in DB
func liquidationOnly(acct *models.TradeAccount) bool {
	accts := strings.Split(env.GetVar("LIQUIDATION_ACCOUNT_NUMBERS"), ",")

	if len(accts) == 0 {
		return true
	}

	if acct.ApexAccount == nil {
		return false
	}

	for _, anum := range accts {
		if strings.EqualFold(anum, *acct.ApexAccount) {
			return true
		}
	}

	return false
}

func (s *orderService) Create(accountID uuid.UUID, o *models.Order) (*models.Order, error) {
	// tx.TX need to be db.DB, just transaction because need to handle 2 tx here.
	// so for here, we need to handle rollback in a right way. Be careful.
	tx := s.tx.Begin()

	defer func() {
		if r := recover(); r != nil {
			if tx != nil {
				tx.Rollback()
			}
			panic(r)
		}
	}()

	acct, err := s.accService.WithTx(tx).ForUpdate().GetByID(accountID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := s.verifyOrder(tx, acct, o); err != nil {
		tx.Rollback()
		return nil, err
	}

	o.SetInitials(acct.LegalName)

	o.Status = enum.OrderAccepted
	o.SubmittedAt = clock.Now()
	o.Account = *acct.ApexAccount

	if err := tx.Create(o).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "idx_client_order_id_account") {
			return nil, gberrors.InvalidRequestParam.WithMsg("client_order_id must be unique")
		}
		return nil, gberrors.InternalServerError.WithMsg("failed to create order").WithError(err)
	}

	req := OrderRequest{
		RequestType: REQ_NEW,
		Order:       o,
	}

	// calculate total equity for pattern day trader marking
	equity, err := s.totalEquity(tx, acct)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// pattern day trader rule
	if equity.LessThan(decimal.New(25000, 0)) && acct.ProtectPatternDayTrader {
		now := clock.Now().In(calendar.NY)

		pdts, err := op.PatternDayTrades(tx, acct, o, now)
		if err != nil {
			tx.Rollback()
			return nil, gberrors.InternalServerError.WithError(fmt.Errorf("failed to calculate pattern day trades"))
		}

		// if the user is already marked, make sure they don't get another
		if acct.PatternDayTrader {
			prevPdts, err := op.PatternDayTrades(tx, acct, nil, now)
			if err != nil {
				tx.Rollback()
				return nil, gberrors.InternalServerError.WithError(fmt.Errorf("failed to calculate pattern day trades"))
			}
			if prevPdts < pdts {
				tx.Rollback()
				return nil, gberrors.Forbidden.WithMsg("account is flagged as a pattern day trader - day trades are restricted")
			}
		}

		// http://www.finra.org/investors/day-trading-margin-requirements-know-rules
		// this count includes the existing pattern day trades, as well as a resulting PDT
		// that could occur due to this new order, hence the check for 4 instead of 3
		if pdts == 4 {
			// protect
			tx.Rollback()
			return nil, gberrors.Forbidden.WithMsg("trade denied due to pattern day trading protection")
		} else if pdts > 4 {
			// mark
			if err = s.accService.WithTx(tx).MarkPatternDayTrader(acct); err != nil {
				tx.Rollback()
				return nil, gberrors.InternalServerError.WithError(err)
			}
		}

	}

	// We'll commit order first in postgres to record we've received order
	// and receive asyncrous update from gotrader.
	// There is also potential for sending order failure to gotrader when rmq is down etc.
	// For that case, ideally we need to have another process which monitor orders
	// status = accepted and took so long. ANd need to ask the order status to fix gateway, but
	// we dicided for now we don't do that because it is pretty edge case.
	if err := tx.Commit().Error; err != nil {
		return nil, errors.Wrap(err, "failed to commit order")
	}

	// released not to call rollback when panic
	tx = nil

	if err := s.submit(acct.IDAsUUID(), req); err != nil {
		// In this case, order might be sent or not. We'll implement garbage collector in different
		// proccess later, but leave it as it for now.
		return nil, fmt.Errorf("failed to submit order (%v)", err)
	}

	tx = s.tx.Begin()

	// Optimistic update to update only accepted state order. Theoretically, this operation
	// has potential to run after update from gotrader.
	if err := tx.Model(&o).Where("status = ?", enum.OrderAccepted).Update("status", enum.OrderNew).Error; err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to update order status")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.Wrap(err, "failed to commit order status update")
	}

	return o, nil
}

func (s *orderService) verifyOrder(tx *gorm.DB, acct *models.TradeAccount, o *models.Order) error {
	balances, err := s.accService.WithTx(tx).GetBalancesByAccount(acct, tradingdate.Last(clock.Now()).MarketOpen())
	if err != nil {
		return err
	}

	o.ClientOrderType = o.Type

	switch o.Type {
	case enum.Market:
		if o.LimitPrice != nil || o.StopPrice != nil {
			return gberrors.InvalidRequestParam.WithMsg("market orders require no stop or limit price")
		}
		if o.Side == enum.Buy {
			if err := models.ToLimit(o, balances.BuyingPower); err != nil {
				return gberrors.Forbidden.WithMsg(err.Error())
			}
		}
	case enum.Limit:
		if o.LimitPrice == nil || o.StopPrice != nil {
			return gberrors.InvalidRequestParam.WithMsg("limit orders require only limit price")
		}
	case enum.Stop:
		if o.StopPrice == nil || o.LimitPrice != nil {
			return gberrors.InvalidRequestParam.WithMsg("stop orders require only stop price")
		}
		if o.Side == enum.Buy {
			if err := models.ToLimit(o, balances.BuyingPower); err != nil {
				return gberrors.Forbidden.WithMsg(err.Error())
			}
		}
	case enum.StopLimit:
		if o.LimitPrice == nil || o.StopPrice == nil {
			return gberrors.InvalidRequestParam.WithMsg("stop limit order requires both stop and limit price")
		}
	default:
		return gberrors.InvalidRequestParam.WithMsg("invalid order type")
	}
	if o.Side == enum.Buy {
		if models.CostBasis(o, false).GreaterThan(balances.BuyingPower) {
			return gberrors.Forbidden.WithMsg("insufficient buying power")
		}
	}
	if !acct.Tradable() {
		return gberrors.Forbidden.WithMsg("account is not authorized to trade")
	}
	if o.Side == enum.Buy && liquidationOnly(acct) {
		tx.Rollback()
		return gberrors.Forbidden.WithMsg("account is restricted to liquidation only")
	}

	return checkAvailableQty(tx, o, acct)
}

func checkAvailableQty(tx *gorm.DB, o *models.Order, acct *models.TradeAccount) error {
	// TODO: short selling comes later...
	qty := decimal.Zero
	switch o.Side {
	case enum.Sell:
		positions := []models.Position{}
		q := tx.Where(
			"account_id = ? AND asset_id = ? AND side = ? AND status = ?",
			acct.ID,
			o.AssetID,
			models.Long,
			models.Open,
		).Find(&positions)

		if q.RecordNotFound() {
			return gberrors.NotFound.WithMsg("position not found")
		}

		if q.Error != nil {
			return gberrors.InternalServerError.WithError(q.Error)
		}

		for _, p := range positions {
			qty = qty.Add(p.Qty)
		}
		orders := []models.Order{}
		q = tx.Where("account = ? and status IN (?)",
			*acct.ApexAccount, enum.OrderOpen).Find(&orders)

		if q.Error != nil && !gorm.IsRecordNotFoundError(q.Error) {
			return gberrors.InternalServerError.WithError(q.Error)
		}

		for _, order := range orders {
			if order.GetSymbol() == o.GetSymbol() && order.Side == o.Side {
				if order.FilledQty != nil {
					qty = qty.Sub(order.Qty.Sub(*order.FilledQty))
				} else {
					qty = qty.Sub(order.Qty)
				}
			}
		}
		if qty.LessThan(o.Qty) {
			return gberrors.Forbidden.WithMsg(
				fmt.Sprintf("insufficient qty (%v < %v)", qty, o.Qty))
		}
	}
	// case db.Buy:
	return nil
}

func (s *orderService) totalEquity(tx *gorm.DB, acct *models.TradeAccount) (*decimal.Decimal, error) {
	balances, err := s.accService.WithTx(tx).GetBalancesByAccount(acct, tradingdate.Current().MarketOpen())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get balances")
	}

	totalValue := balances.Cash

	srv := s.posService.WithTx(tx)

	positions, err := srv.List(acct.IDAsUUID())
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		totalValue = totalValue.Add(pos.MarketValue)
	}

	return &totalValue, nil
}
