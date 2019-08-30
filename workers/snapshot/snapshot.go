package snapshot

import (
	"fmt"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gobroker/utils/txlevel"
	"github.com/alpacahq/gobroker/workers/common"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	pricefmt "github.com/alpacahq/gopaca/price"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

var lastDate *tradingdate.TradingDate

type snapshotWorker struct {
	done chan struct{}
}

var worker *snapshotWorker

func Work() {
	if worker == nil {
		worker = &snapshotWorker{done: make(chan struct{}, 1)}
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

	now := clock.Now().In(calendar.NY)
	// run at 8:30 AM EST (5:30 PST)
	on := now.Truncate(24 * time.Hour).Add(time.Hour * 8).Add(time.Minute * 30)
	if !now.After(on) {
		return
	}

	lastTradeDay := tradingdate.Last(on)

	if lastDate != nil && lastDate.String() == lastTradeDay.String() {
		// Already done
		return
	}

	worker.processSnapshot(lastTradeDay)
	lastDate = &lastTradeDay
}

func isIn(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// computePortfolioValue will be replaced with SoD one
func computeDayPL(tx *gorm.DB, acc *models.Account, target tradingdate.TradingDate) (decimal.Decimal, error) {
	positions := []models.Position{}
	if err := tx.Raw(`
		SELECT
		  *
		FROM
		  positions
		WHERE
		  account_id = ? AND
			status != 'split' AND
		  (
		    (
		      entry_timestamp < ? AND
		      (exit_timestamp IS NULL OR exit_timestamp >= ?)
		    ) OR
		    (
		      entry_timestamp >= ? AND
		      entry_timestamp < ?
		    )
		  )
		;
	`, acc.ID, target.MarketOpen(), target.MarketOpen(), target.MarketOpen(), target.Next().MarketOpen()).Scan(&positions).Error; err != nil {
		log.Panic("start of day database error", "error", err)
	}

	if len(positions) == 0 {
		return decimal.Zero, nil
	}

	currentClosingSymbols := []string{}
	for _, p := range positions {
		if p.ExitTimestamp == nil || p.ExitTimestamp.After(target.MarketClose()) {
			asset := assetcache.GetByID(p.AssetID)
			if asset == nil {
				err := fmt.Errorf("failed to lookup symbol from assetID (%s)", p.AssetID)
				log.Panic("start of day error", "error", err)
			}
			symbol := asset.Symbol
			if !isIn(symbol, currentClosingSymbols) {
				currentClosingSymbols = append(currentClosingSymbols, symbol)
			}
		}
	}

	lastClosingSymbols := []string{}
	for _, p := range positions {
		if p.EntryTimestamp.Before(target.MarketOpen()) {
			asset := assetcache.GetByID(p.AssetID)
			if asset == nil {
				err := fmt.Errorf("failed to lookup symbol from assetID (%v)", p.AssetID)
				log.Panic("start of day error", "error", err)
			}
			symbol := asset.Symbol
			if !isIn(symbol, lastClosingSymbols) {
				lastClosingSymbols = append(lastClosingSymbols, symbol)
			}
		}
	}

	until := target.MarketClose().Add(-1 * time.Millisecond)
	clp, err := price.GetLatest(currentClosingSymbols, &until)
	if err != nil {
		panic(err)
	}

	llp, err := price.LastDayClosing(lastClosingSymbols, &target)
	if err != nil {
		panic(err)
	}

	value := decimal.Zero

	for _, p := range positions {
		asset := assetcache.GetByID(p.AssetID)
		if asset == nil {
			err := fmt.Errorf("failed to lookup symbol from assetID (%v)", p.AssetID)
			log.Panic("start of day error", "error", err)
		}
		symbol := asset.Symbol

		var o, c decimal.Decimal
		if p.EntryTimestamp.Before(target.MarketOpen()) {
			o = pricefmt.FormatFloat64ForCalc(llp[symbol].Price)
		} else {
			o = p.EntryPrice
		}
		if p.ExitTimestamp == nil || p.ExitTimestamp.After(target.MarketClose()) {
			c = pricefmt.FormatFloat64ForCalc(clp[symbol].Price)
		} else {
			c = *p.ExitPrice
		}
		value = value.Add(c.Sub(o).Mul(p.Qty))
	}

	// subtract fee for that day. Use executions not order.fee. Because order.fee might not be idempotent for overnight orders.
	fee := struct{ Fee decimal.Decimal }{}
	if err := tx.Model(&models.Execution{}).Select("SUM(fee_misc + fee_sec + fee1 + fee2 + fee3 + fee4 + fee5) as fee").
		Where(
			"account = ? AND transaction_time >= ? AND transaction_time < ?",
			acc.ApexAccount, target.MarketOpen(), target.Next().MarketOpen()).
		Scan(&fee).Error; err != nil {
		log.Panic("start of day database error", "error", err)
	}
	value = value.Sub(fee.Fee)

	// Add dividends for that day
	var dividends []models.Dividend
	if err := tx.Where("account_id = ? AND pay_date = ?", acc.ID, target.String()).Find(&dividends).Error; err != nil {
		log.Panic("start of day database error", "error", err)
	}

	for _, d := range dividends {
		if d.DividendInterest != nil {
			value = value.Add(*d.DividendInterest)
		} else {
			log.Error("start of day error", "error", fmt.Errorf("dividend_interest is missing"))
		}
	}

	return value, nil
}

type TotalPL struct {
	ProfitLoss decimal.Decimal
}

func trackFrom(acc *models.Account) tradingdate.TradingDate {
	// snapshot will be created on the next start of trading day.
	if calendar.IsMarketDay(acc.CreatedAt.In(calendar.NY)) {
		t, _ := tradingdate.New(acc.CreatedAt.In(calendar.NY))
		return t.Next()
	} else {
		return tradingdate.Last(acc.CreatedAt.In(calendar.NY)).Next()
	}
}

var (
	ErrCashUnavailable    = errors.New("sod cash is not available")
	ErrPendingApexAccount = errors.New("apex account is not created")
)

/* With SoD file, we use last trading day's portfolio value. This is represented by OvernightBuyingPowerCalculated
 * and EOD market value of the positions at that day. OvernightBuyingPowerCalculated which is comes from SoD file.
 * It is equal to cash now as we don't support more than 1x margin.
 */
func ComputeBasisWithSoD(tx *gorm.DB, acct *models.Account, target tradingdate.TradingDate) (*decimal.Decimal, error) {
	if acct.ApexAccount == nil {
		return nil, ErrPendingApexAccount
	}

	// t-1 trading days position market value + cash value
	{
		ok, err := txlevel.Repeatable(tx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("requires repeatable read tx")
		}
	}

	// Need to use previous trading day's cash, because we need previous day's cash value.
	// cash value is end of day value, so we don't need to subtract intraday transfers and position changes.
	var cash models.Cash
	prev := target.Prev()

	// TODO: cash needs to including pending transfer up to instant deposit limit
	q := tx.Where("account_id = ? AND date = ?", acct.ID, prev.String()).Find(&cash)
	if q.RecordNotFound() {
		return nil, ErrCashUnavailable
	}

	if q.Error != nil {
		return nil, errors.Wrap(q.Error, "database error")
	}

	// Get prev trading day of target day's position market value on EoD
	svc := gbreg.Services.Position().WithTx(tx)
	positionValue, err := svc.MarketValueAt(acct.IDAsUUID(), prev)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate market value")
	}

	value := cash.Value.Add(positionValue)

	// TODO: enable for instant deposit
	// // include pending transfers up to instant deposit limit
	// if value.LessThan(constants.InstantDepositLimit) {
	// 	incomingTransfers := []models.Transfer{}
	// 	if err := tx.Where(
	// 		`account_id = ? AND
	// 		direction = ? AND
	// 		status NOT IN (?)
	// 		AND batch_processed_at IS NULL`,
	// 		acct.ID, apex.Incoming, []apex.TransferStatus{
	// 			apex.TransferRejected,
	// 			apex.TransferCanceled,
	// 			apex.TransferReturned,
	// 			apex.TransferVoid,
	// 			apex.TransferStopPayment,
	// 		}).Find(&incomingTransfers).Error; err != nil {
	// 		return nil, errors.Wrap(err, "database error")
	// 	}

	// 	for _, transfer := range incomingTransfers {
	// 		if value.LessThan(constants.InstantDepositLimit) {
	// 			value = cash.Value.Add(transfer.Amount)
	// 			if value.GreaterThan(constants.InstantDepositLimit) {
	// 				value = constants.InstantDepositLimit
	// 				break
	// 			}
	// 		}
	// 	}
	// }

	return &value, nil
}

func computeBasisDev(tx *gorm.DB, acct *models.Account, target tradingdate.TradingDate) (*decimal.Decimal, error) {
	// SoD file is not available on dev environment, so we are using transfers and profits.
	transfers := []models.Transfer{}
	err := tx.Raw(`
	SELECT
	  *
	FROM
	  transfers
	WHERE
	  account_id = ? AND
		status = 'COMPLETE' AND
		updated_at < ?
	;
`, acct.ID, target.MarketClose()).Scan(&transfers).Error

	if err != nil {
		return nil, err
	}

	var totalPL TotalPL
	err = tx.Raw(`
	SELECT
		SUM(profit_loss) as profit_loss
	FROM
		day_pl_snapshots
	WHERE
		account_id = ? AND
		date < ?
`, acct.ID, target.String()).Scan(&totalPL).Error

	if err != nil {
		return nil, err
	}

	value := decimal.Zero

	for _, t := range transfers {
		if t.Direction == apex.Incoming {
			value = value.Add(t.Amount)
		} else {
			value = value.Sub(t.Amount)
		}
	}

	value = value.Add(totalPL.ProfitLoss)

	return &value, nil
}

func computeBasis(tx *gorm.DB, acct *models.Account, target tradingdate.TradingDate) (*decimal.Decimal, error) {
	if utils.Dev() {
		return computeBasisDev(tx, acct, target)
	}

	return ComputeBasisWithSoD(tx, acct, target)
}

func ProcessSnapshot(target tradingdate.TradingDate) {
	// End of Day value is to be calculated for the target day.
	if worker == nil {
		worker = &snapshotWorker{}
	}
	worker.processSnapshot(target)
}

func (w *snapshotWorker) processSnapshot(target tradingdate.TradingDate) {
	accounts := []models.Account{}

	if err := db.DB().Find(&accounts).Error; err != nil {
		log.Panic("start of day database error", "error", err)
	}

	for _, a := range accounts {
		firstAt := trackFrom(&a)

		if target.Before(firstAt) {
			continue
		}

		tx := db.RepeatableRead()

		// compute the previous day's profit/loss
		pl, err := computeDayPL(tx, &a, target)
		if err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "error", err)
		}

		// compute the basis
		basis, err := computeBasis(tx, &a, target)
		if err != nil {
			switch {
			case err == ErrCashUnavailable:
				fallthrough
			case err == ErrPendingApexAccount:
				tx.Rollback()
				continue
			default:
				tx.Rollback()
				log.Panic("start of day database error", "error", err)
			}
		}

		var snapshot models.DayPLSnapshot
		if err := tx.Where("account_id = ? AND date = ?", a.ID, target.String()).Find(&snapshot).Error; err != nil {
			switch {
			case gorm.IsRecordNotFoundError(err):
				snapshot.AccountID = a.ID
				snapshot.ProfitLoss = pl
				snapshot.Basis = *basis
				snapshot.Date = target.String()
				if err := tx.Create(&snapshot).Error; err != nil {
					tx.Rollback()
					log.Panic("start of day database error", "error", err)
				}
			default:
				tx.Rollback()
				log.Panic("start of day database error", "error", err)
			}
		} else {
			if err := tx.Model(&snapshot).Updates(models.DayPLSnapshot{ProfitLoss: pl, Basis: *basis}).Error; err != nil {
				tx.Rollback()
				log.Panic("start of day database error", "error", err)
			}
		}

		if err := tx.Commit().Error; err != nil {
			log.Panic("start of day database error", "error", err)
		}
	}
}
