package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gbreg"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDMandatoryAction struct {
	Firm                string                  `sql:"type:text"`
	Cusip               string                  `sql:"type:text"`
	CusipOld            string                  `sql:"type:text"`
	Symbol              string                  `sql:"type:text"`
	SymbolOld           string                  `sql:"type:text"`
	ShortDescription    string                  `sql:"type:text"`
	ShortDescriptionOld string                  `sql:"type:text"`
	ISIN                string                  `sql:"type:text"`
	ExpirationDate      *string                 `sql:"type:date"`
	ProcessDate         *string                 `sql:"type:date"`
	ToMarket            string                  `sql:"type:text"`
	FromMarket          string                  `sql:"type:text"`
	CountryCode         string                  `sql:"type:text"`
	CountryCodeOld      string                  `sql:"type:text"`
	StockFactor         *decimal.Decimal        `gorm:"type:decimal"`
	CashFactor          *decimal.Decimal        `gorm:"type:decimal"`
	PayableDate         *string                 `sql:"type:date"`
	SettlementDate      *string                 `sql:"type:date"`
	LastChangeDate      *string                 `sql:"type:date"`
	Action              enum.SoDCorporateAction `sql:"type:text"`
	ActionMessage       string                  `sql:"type:text"`
	RecordDate          *string                 `sql:"type:date"`
}

type MandatoryActionReport struct {
	actions []SoDMandatoryAction
}

func (ma *MandatoryActionReport) ExtCode() string {
	return "EXT235"
}

func (ma *MandatoryActionReport) Delimiter() string {
	return "|"
}

func (ma *MandatoryActionReport) Header() bool {
	return false
}

func (ma *MandatoryActionReport) Extension() string {
	return "txt"
}

func (ma *MandatoryActionReport) Value() reflect.Value {
	return reflect.ValueOf(ma.actions)
}

func (ma *MandatoryActionReport) Append(v interface{}) {
	ma.actions = append(ma.actions, v.(SoDMandatoryAction))
}

// Sync goes through the mandatory corporate actions in the file,
// and cancels any open orders on the affected equities. Any errors
// are stored in the batch_errors table in DB.
func (ma *MandatoryActionReport) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}

	// Apex sometimes (inconsistently) includes lower case symbol/cusip but
	// our databasee is in upper case.
	for i, _ := range ma.actions {
		ma.actions[i].Symbol = strings.ToUpper(ma.actions[i].Symbol)
		ma.actions[i].SymbolOld = strings.ToUpper(ma.actions[i].SymbolOld)
		ma.actions[i].Cusip = strings.ToUpper(ma.actions[i].Cusip)
		ma.actions[i].CusipOld = strings.ToUpper(ma.actions[i].CusipOld)
	}

	// aggregate separate records into one using cusip old
	// See comment on AggedAction, too.
	aggs := map[string]*AggedAction{}
	for i, _ := range ma.actions {
		action := &ma.actions[i]
		cusipOld := action.CusipOld

		if _, ok := aggs[cusipOld]; ok {
			aggs[cusipOld].actions = append(aggs[cusipOld].actions, action)
		} else {
			agged := &AggedAction{
				SoDMandatoryAction: *action,
				actions:            []*SoDMandatoryAction{action},
			}
			aggs[cusipOld] = agged
		}
	}

	for _, agged := range aggs {
		found, err := ma.findAsset(asOf, agged)
		if err != nil {
			errors = append(errors, *err)
			continue
		}

		if !found {
			// we don't have any matching asset, skip it (e.g. non-equity symbols)
			continue
		}

		// update the asset
		if err := ma.handleAsset(asOf, agged); err != nil {
			errors = append(errors, *err)
			continue
		}

		// find and cancel open orders
		if err := ma.handleOrders(asOf, agged); err != nil {
			errors = append(errors, *err)
		}

		// find and mark positions with splits
		if err := ma.handlePositions(asOf, agged); err != nil {
			errors = append(errors, *err)
		}

		// store the corporate action in the DB
		if err := ma.storeAllActions(asOf, agged); err != nil {
			log.Panic("start of day database error", "file", ma.ExtCode(), "error", err)
		}

	}

	StoreErrors(errors)

	return uint(len(ma.actions) - len(errors)), uint(len(errors))
}

func (ma *MandatoryActionReport) genError(asOf time.Time, action SoDMandatoryAction, err error) *models.BatchError {
	log.Error("start of day error", "file", ma.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":  err.Error(),
		"action": action,
	})

	return &models.BatchError{
		ProcessDate:               asOf.Format("2006-01-02"),
		FileCode:                  ma.ExtCode(),
		PrimaryRecordIdentifier:   string(action.Action),
		SecondaryRecordIdentifier: action.Symbol,
		Error:                     buf,
	}
}

func (ma *MandatoryActionReport) storeAllActions(asOf time.Time, agged *AggedAction) error {
	for _, action := range agged.actions {
		if err := ma.storeAction(asOf, *action, &agged.asset); err != nil {
			return err
		}
	}
	return nil
}

func (ma *MandatoryActionReport) storeAction(asOf time.Time, action SoDMandatoryAction, asset *models.Asset) error {
	a := &models.CorporateAction{
		AssetID:     asset.IDAsUUID(),
		Type:        enum.CorporateActionTypeFromSoD(action.Action),
		Date:        asOf.Format("2006-01-02"),
		StockFactor: action.StockFactor,
		CashFactor:  action.CashFactor,
	}

	tx := db.RepeatableRead()

	if err := tx.Save(a).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (ma *MandatoryActionReport) handleAsset(
	asOf time.Time,
	agged *AggedAction) *models.BatchError {

	if agged.needUpdateAsset() {
		ma.updateAsset(&agged.asset, agged.SoDMandatoryAction)
	}

	return nil
}

func (ma *MandatoryActionReport) updateAsset(asset *models.Asset, action SoDMandatoryAction) {
	// update the cusip to be latest
	if action.Cusip != "" {
		asset.CUSIP = action.Cusip
	}

	// persist the old cusip
	if action.CusipOld != "" {
		asset.CUSIPOld = action.CusipOld
	}

	// update the symbol to be latest
	if action.Symbol != "" {
		asset.Symbol = action.Symbol
	}

	// persist the old symbol
	if action.SymbolOld != "" {
		asset.SymbolOld = action.SymbolOld
	}

	tx := db.RepeatableRead()

	err := tx.Save(asset).Error

	if err != nil {
		tx.Rollback()
	} else {
		err = tx.Commit().Error
	}

	if err != nil {
		log.Panic(
			"start of day database error",
			"file", ma.ExtCode(),
			"cusip_old", action.CusipOld,
			"cusip", action.Cusip,
			"symbol_old", action.SymbolOld,
			"symbol", action.Symbol,
			"error", err)
	}
}

func (ma *MandatoryActionReport) handleOrders(
	asOf time.Time,
	agged *AggedAction) (bErr *models.BatchError) {

	orders := []models.Order{}

	// find open orders for the asset
	q := db.DB().
		Where("asset_id = ? AND status NOT IN (?)",
			agged.asset.ID, []enum.OrderStatus{
				enum.OrderCanceled,
				enum.OrderRejected,
				enum.OrderExpired,
				enum.OrderFilled}).
		Find(&orders)

	if q.Error != nil {
		log.Panic("start of day database error", "file", ma.ExtCode(), "error", q.Error)
	}

	if len(orders) == 0 {
		// no open orders, let's continue
		return nil
	}

	// cancel the open orders
	srv := gbreg.Services.Order()
	for _, order := range orders {
		acct := &models.Account{}
		q := db.DB().Where("apex_account = ?", order.Account).Find(acct)

		if q.RecordNotFound() {
			bErr = ma.genError(asOf, agged.SoDMandatoryAction, fmt.Errorf("account not found"))
			return
		}

		if q.Error != nil {
			log.Panic("start of day database error", "file", ma.ExtCode(), "error", q.Error)
		}

		tx := db.RepeatableRead()

		if err := srv.WithTx(tx).Cancel(acct.IDAsUUID(), order.IDAsUUID()); err != nil {
			tx.Rollback()
			bErr = ma.genError(asOf, agged.SoDMandatoryAction, err)
			return
		}

		tx.Commit()
	}
	return nil
}

// mark positions for split
func (ma *MandatoryActionReport) handlePositions(
	asOf time.Time,
	agged *AggedAction) (bErr *models.BatchError) {

	if !agged.needMarkPosition() {
		return
	}

	positions := []models.Position{}
	asset := agged.asset
	// find open positions for the asset
	q := db.DB().Where("asset_id = ? AND status = ?",
		asset.ID, models.Open).Find(&positions)

	if q.Error != nil {
		if q.Error != nil {
			log.Panic("start of day database error", "file", ma.ExtCode(), "error", q.Error)
		}
	}

	if len(positions) == 0 {
		return nil
	}

	tx := db.RepeatableRead()

	// mark them for split
	for i := range positions {
		if err := tx.
			Model(&positions[i]).
			Update("marked_for_split_at", asOf.Format("2006-01-02")).Error; err != nil {

			tx.Rollback()
			log.Panic("start of day database error", "file", ma.ExtCode(), "error", q.Error)
		}
	}

	tx.Commit()

	return
}

func containsDigits(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// findAsset finds asset for AggedAction
func (ma *MandatoryActionReport) findAsset(
	asOf time.Time, agged *AggedAction) (found bool, bErr *models.BatchError) {

	q := db.DB().Where("cusip = ?", agged.CusipOld).First(&agged.asset)

	if q.RecordNotFound() {
		// very much expected, do nothing.
		return false, nil
	}

	if q.Error != nil {
		log.Panic("start of day database error", "file", ma.ExtCode(), "error", q.Error)
	}

	// double check
	if agged.SymbolOld != agged.asset.Symbol &&
		// symbol_old like Y003680 isn't very meaningful (likely already delisted)
		!containsDigits(agged.SymbolOld) {
		log.Error("symbol mismatch",
			"cusip", agged.CusipOld,
			"symbol", agged.asset.Symbol,
			"symbol_old", agged.SymbolOld)

		return true, ma.genError(
			asOf, agged.SoDMandatoryAction,
			fmt.Errorf("symbol mistmatch: cusip = %s, symbol = %s, symbol_old = %s",
				agged.CusipOld, agged.asset.Symbol, agged.SymbolOld),
		)
	}

	return true, nil
}

// AggedAction is a group of SodMandatoryAction
// This is to reflect 1:N between asset and each action that may be more than one per day.
// For example, one symbol could have reverse split and cusip change.
// After 6 months so far, it seems we can rely on CusipOld as a key and
// each record within the same asset has the same assset information regardless of the action.
type AggedAction struct {
	// Just piggy-back on the Sod definition, but Action/ActionMessage are
	// not useful in this embedded field. Use actions for each different action.
	SoDMandatoryAction

	actions []*SoDMandatoryAction
	asset   models.Asset
}

func (agg *AggedAction) needUpdateAsset() bool {
	for _, action := range agg.actions {
		switch action.Action {
		case enum.SoDSymbolChange:
			fallthrough
		case enum.SoDCusipChange:
			return true
		}
	}
	return false
}

func (agg *AggedAction) needMarkPosition() bool {
	for _, action := range agg.actions {
		switch action.Action {
		case enum.SoDCashMerger:
			fallthrough
		case enum.SoDStockMerger:
			fallthrough
		case enum.SoDStockSplit:
			fallthrough
		case enum.SoDReverseSplit:
			return true
		}
	}
	return false
}
