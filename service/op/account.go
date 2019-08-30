package op

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/pkg/errors"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/shopspring/decimal"
)

type trades []time.Time

func (t trades) Len() int {
	return len(t)
}

func (t trades) Less(i, j int) bool {
	return t[i].Before(t[j])
}

func (t trades) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

type orderHistory struct {
	Orders     []models.Order
	Executions []models.Execution
}

type orderHistoryBySymbol struct {
	data map[string]*orderHistory
}

func (hbs *orderHistoryBySymbol) GetOrInit(symbol string) *orderHistory {
	if hbs.data == nil {
		hbs.data = make(map[string]*orderHistory)
	}
	history, ok := hbs.data[symbol]
	if !ok {
		history = &orderHistory{}
		hbs.data[symbol] = history
	}
	return history
}

// GetAccountByID returns the account corresponding to the given ID. It is
// important to note that the query does a SELECT FOR <queryOption>, and as a result
// locks the account row until the transaction is committed, if FOR UPDATE is set.
func GetAccountByID(tx *gorm.DB, accountID uuid.UUID, queryOption *string) (*models.Account, error) {
	a := &models.Account{}

	q := tx.
		Where("id = ?", accountID.String()).
		Preload("Owners").
		Preload("Owners.Details", "replaced_by IS NULL")

	// only set the query option if it is supplied
	if queryOption != nil {
		q = q.Set("gorm:query_option", *queryOption)
	}

	q = q.First(a)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("account not found for %v", accountID.String()))
	}

	if q.Error != nil {
		return nil, q.Error
	}

	return a, nil
}

// GetAccountByApexAccount query account by on apex account id
func GetAccountByApexAccount(tx *gorm.DB, apexAccount string, queryOption *string) (*models.Account, error) {
	a := &models.Account{}

	q := tx.
		Where("apex_account = ?", apexAccount).
		Preload("Owners").
		Preload("Owners.Details", "replaced_by IS NULL")

	// only set the query option if it is supplied
	if queryOption != nil {
		q = q.Set("gorm:query_option", *queryOption)
	}

	q = q.First(a)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("account not found for apex_account %v", apexAccount))
	}

	if q.Error != nil {
		return nil, q.Error
	}

	return a, nil
}

func GetAccountByCognitoID(tx *gorm.DB, cognitoID uuid.UUID, queryOption *string) (*models.Account, error) {
	a := &models.Account{}

	q := tx.
		Where("cognito_id = ?", cognitoID.String()).
		Preload("Owners").
		Preload("Owners.Details")

	// only set the query option if it is supplied
	if queryOption != nil {
		q = q.Set("gorm:query_option", *queryOption)
	}

	q = q.First(a)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("account not found for cognito_id %v", cognitoID))
	}

	if q.Error != nil {
		return nil, q.Error
	}

	return a, nil
}

// This method probably does not hold up to DST...
// Should investigate and leave clarifying comment
func dayOf(x time.Time) time.Time {
	return x.In(calendar.NY).Truncate(24 * time.Hour)
}

// counts day trades for a single symbol
// pass executions from the last 5 trading days
func confirmedDayTrades(executions []models.Execution) int {
	if len(executions) <= 1 {
		return 0
	}
	count := 0
	last := executions[0]
	lastDay := dayOf(last.TransactionTime)
	for _, ex := range executions[1:] {
		exDay := dayOf(ex.TransactionTime)
		if exDay == lastDay && last.Side == enum.Buy && ex.Side == enum.Sell {
			count++
		}
		last = ex
		lastDay = exDay
	}
	return count
}

// counts potential day trades for a single symbol
func potentialDayTrades(lastExToday *models.Execution, orders []models.Order) int {
	buys, sells := 0, 0
	for _, order := range orders {
		// It would be more correct to count the total number of pending *shares* ordered,
		// because each pending share can theoretically result in a separate execution.
		// But as of 2018-12-12, for the sake of permissiveness we are only counting orders.
		switch order.Side {
		case enum.Buy:
			buys++
		case enum.Sell:
			sells++
		}
	}
	// When calculating potential PDTs, we only care about the worst case.
	// There can only be as many day trades as the lesser number of sells or buys.
	// e.g. if you only have 1 sell, you can only have 1 DT no matter how many buys there are.
	potential := sells
	if buys < sells {
		potential = buys
	}
	// the addendum to the above rule is if the last execution is a buy,
	// and you have more sells than buys, one of the extra sells could be a day trade if it fills first.
	if sells > buys && lastExToday != nil && lastExToday.Side == enum.Buy {
		potential++
	}
	return potential
}

// PatternDayTrades calculates the account's weekly (past 5 trading days) running count
// of day trades using executions and pending orders
func PatternDayTrades(tx *gorm.DB, a *models.TradeAccount, order *models.Order, now time.Time) (int, error) {
	orders := []models.Order{}
	execs := []models.Execution{}
	today := dayOf(now)

	windowStart := today
	for i := 0; i < 5; i++ {
		windowStart = calendar.PrevClose(windowStart)
	}

	if err := tx.
		Where("account = ? AND status IN (?)", a.ApexAccount, enum.OrderOpen).
		Order("symbol").
		Find(&orders).Error; err != nil {
		return 0, err
	}

	if err := tx.
		Where("account = ? AND transaction_time >= ? AND type IN (?)",
			a.ApexAccount,
			calendar.MarketOpen(windowStart),
			[]enum.ExecutionType{enum.ExecutionFill, enum.ExecutionPartialFill}).
		Order("symbol, transaction_time").
		Find(&execs).Error; err != nil {
		return 0, err
	}

	if len(orders) == 0 && len(execs) == 0 {
		return 0, nil
	}

	// append the a copy of the new order to determine
	// if it's going to trigger a new PDT
	if order != nil {
		o := *order
		orders = append(orders, o)
	}

	// group the orders and executions by symbol
	histories := orderHistoryBySymbol{}

	for _, order := range orders {
		history := histories.GetOrInit(order.GetSymbol())
		history.Orders = append(history.Orders, order)
	}

	for _, exec := range execs {
		history := histories.GetOrInit(exec.Symbol)
		history.Executions = append(history.Executions, exec)
	}

	dayTrades := 0

	for _, h := range histories.data {
		lenX := len(h.Executions)
		var lastExec *models.Execution
		if lenX > 0 && dayOf(h.Executions[lenX-1].TransactionTime) == today {
			lastExec = &h.Executions[lenX-1]
		}
		dayTrades += potentialDayTrades(lastExec, h.Orders)
		dayTrades += confirmedDayTrades(h.Executions)
	}

	return dayTrades, nil
}

// GetAccountTradingDate returns trading date based on the account.cash changes.
// In some cases, we need to atomically switch state based on cash value change, and this is helper
// for these features.
func GetAccountTradingDate(tx *gorm.DB, accountID uuid.UUID) (*tradingdate.TradingDate, error) {
	// Get last cash date to detect SoD account.cash update.
	var date tradingdate.TradingDate
	var cash models.Cash
	if err := tx.Where("account_id = ?", accountID).Order("date desc").First(&cash).Error; err != nil {
		switch {
		case gorm.IsRecordNotFoundError(err):
			// in this case, treat current date as tradingdate.
			cur := tradingdate.Current()
			return &cur, nil
		default:
			return nil, errors.Wrap(err, "failed to get cash")
		}
	} else {
		d, err := tradingdate.NewFromDate(cash.Date.Year, cash.Date.Month, cash.Date.Day)
		if err != nil {
			return nil, errors.Wrap(err, "no tradingday cash found")
		}
		date = *d
	}

	cur := date.Next()

	return &cur, nil
}

// GetAccountBalances returns the intraday cash and buying power information for the
// given account, using the provided DB transaction and market open time.
func GetAccountBalances(tx *gorm.DB, a *models.TradeAccount) (*models.IntradayBalances, error) {
	icw := a.CashWithdrawable
	ic := a.Cash
	ibp := a.Cash

	// TODO: enable for when instant deposit is ready
	// // add buying power for pending incoming
	// // transfers up to $1000 (instant deposit)
	// if ibp.LessThan(constants.InstantDepositLimit) {
	// 	incomingTransfers := []Transfer{}
	// 	if err := tx.Where(
	// 		`account_id = ? AND
	// 		direction = ? AND
	// 		status NOT IN (?)
	// 		AND batch_processed_at IS NULL`,
	// 		a.ID, apex.Incoming, []apex.TransferStatus{
	// 			apex.TransferRejected,
	// 			apex.TransferCanceled,
	// 			apex.TransferReturned,
	// 			apex.TransferVoid,
	// 			apex.TransferStopPayment,
	// 		}).Find(&incomingTransfers).Error; err != nil {
	// 		return nil, err
	// 	}

	// 	for _, transfer := range incomingTransfers {
	// 		// buying power
	// 		if ibp.LessThan(constants.InstantDepositLimit) {
	// 			ibp = ibp.Add(transfer.Amount)
	// 			if ibp.GreaterThan(constants.InstantDepositLimit) {
	// 				ibp = constants.InstantDepositLimit
	// 				break
	// 			}
	// 		}
	// 	}

	// 	for _, transfer := range incomingTransfers {
	// 		// cash
	// 		if ic.LessThan(constants.InstantDepositLimit) {
	// 			ic = ic.Add(transfer.Amount)
	// 			if ic.GreaterThan(constants.InstantDepositLimit) {
	// 				ic = constants.InstantDepositLimit
	// 				break
	// 			}
	// 		}
	// 	}
	// }

	date, err := GetAccountTradingDate(tx, a.IDAsUUID())
	if err != nil {
		return nil, err
	}

	// subtract pending outgoing transfers [ICW, IC, IBP]
	{
		outgoingTransfers := []models.Transfer{}
		if err := tx.Where(
			`account_id = ? AND
			direction = ? AND
			status NOT IN (?)
			AND batch_processed_at IS NULL`,
			a.ID, apex.Outgoing, []apex.TransferStatus{
				apex.TransferRejected,
				apex.TransferCanceled,
				apex.TransferReturned,
				apex.TransferVoid,
				apex.TransferStopPayment,
			}).Find(&outgoingTransfers).Error; err != nil {
			return nil, err
		}

		for _, transfer := range outgoingTransfers {
			icw = icw.Sub(transfer.Amount)
			ic = ic.Sub(transfer.Amount)
			ibp = ibp.Sub(transfer.Amount)
		}
	}

	// subtract open orders [ICW, IBP]
	{
		openOrders := []models.Order{}
		if err := tx.Where(
			"account = ? AND status = ? AND side = ?",
			a.ApexAccount,
			enum.OrderNew,
			enum.Buy).Find(&openOrders).Error; err != nil {
			return nil, err
		}

		for _, order := range openOrders {
			costBasis := models.CostBasis(&order, true)

			icw = icw.Sub(costBasis)
			ibp = ibp.Sub(costBasis)
		}
	}

	// subtract new entries [ICW, IC, IBP]
	{
		newEntries := []models.Position{}
		if err := tx.Where(
			"account_id = ? AND status != ? AND entry_timestamp >= ?",
			a.ID, models.Split, date.SessionOpen()).Find(&newEntries).Error; err != nil {
			return nil, err
		}

		for _, entry := range newEntries {
			val := entry.EntryPrice.Mul(entry.Qty)

			icw = icw.Sub(val)
			ic = ic.Sub(val)
			ibp = ibp.Sub(val)
		}
	}

	// add new exits [IC, IBP]
	{
		newExits := []models.Position{}

		if err := tx.Where(
			"account_id = ? AND status != ? AND exit_timestamp >= ?",
			a.ID, models.Split, date.SessionOpen()).Find(&newExits).Error; err != nil {
			return nil, err
		}

		for _, exit := range newExits {
			val := exit.Qty.Mul(*exit.ExitPrice)

			ic = ic.Add(val)
			ibp = ibp.Add(val)
		}
	}

	return &models.IntradayBalances{
		CashWithdrawable: decimal.Max(icw, decimal.Zero),
		Cash:             ic,
		BuyingPower:      ibp,
	}, nil
}
