package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDAccountType string

const (
	SoDAccountTypeFutures1            SoDAccountType = "&"
	SoDAccountTypeBrokerDealer        SoDAccountType = "0"
	SoDAccountTypeCash                SoDAccountType = "1"
	SoDAccountTypeGeneralMargin       SoDAccountType = "2"
	SoDAccountTypeShort               SoDAccountType = "3"
	SoDAccountTypeSpecialSubscription SoDAccountType = "4"
	SoDAccountTypeFutures2            SoDAccountType = "5"
	SoDAccountTypeFunding             SoDAccountType = "6"
	SoDAccountTypeRestricted          SoDAccountType = "7"
	SoDAccountTypeDivIntPayable       SoDAccountType = "8"
	SoDAccountTypeDivIntReceivable    SoDAccountType = "9"
	SoDAccountTypeCreditInterest      SoDAccountType = "C"
	SoDAccountTypeTefraEscrow         SoDAccountType = "E"
	SoDAccountTypeMoneyMarket         SoDAccountType = "M"
	SoDAccountTypeNonConvertibleBonds SoDAccountType = "N"
)

type SoDSecurityType string

const (
	SoDSecurityTypeFuture                                SoDSecurityType = "&"
	SoDSecurityTypeSSFuture                              SoDSecurityType = "@"
	SoDSecurityTypeSSCFD                                 SoDSecurityType = "~"
	SoDSecurityTypeEquityFedFunds                        SoDSecurityType = "0"
	SoDSecurityTypeEquityUnit                            SoDSecurityType = "1"
	SoDSecurityTypeEquityCommodity                       SoDSecurityType = "2"
	SoDSecurityTypeEquityRight                           SoDSecurityType = "3"
	SoDSecurityTypeEquityWarrant                         SoDSecurityType = "4"
	SoDSecurityTypeOptionCall                            SoDSecurityType = "5"
	SoDSecurityTypeOptionPut                             SoDSecurityType = "6"
	SoDSecurityTypeEquityUnderlyingIndexOption           SoDSecurityType = "7"
	SoDSecurityTypeEquityUnderlyingCurrencyOption        SoDSecurityType = "8"
	SoDSecurityTypeEquityUnderlyingForeignCurrencyOption SoDSecurityType = "9"
	SoDSecurityTypeEquityCommonStock                     SoDSecurityType = "A"
	SoDSecurityTypeEquityPreferredStock                  SoDSecurityType = "B"
	SoDSecurityTypeEquityMutualFund                      SoDSecurityType = "C"
	SoDSecurityTypeEquityMiscStock                       SoDSecurityType = "F"
	SoDSecurityTypeEquityTaxableTrust                    SoDSecurityType = "G"
	SoDSecurityTypeEquityNonTaxableTrust                 SoDSecurityType = "H"
	SoDSecurityTypeBondUnit                              SoDSecurityType = "I"
	SoDSecurityTypeBondCorporate                         SoDSecurityType = "J"
	SoDSecurityTypeBondMunicipalNote                     SoDSecurityType = "K"
	SoDSecurityTypeBondMunicipal                         SoDSecurityType = "L"
	SoDSecurityTypeBondGovernment                        SoDSecurityType = "M"
	SoDSecurityTypeBondMoneyMarket                       SoDSecurityType = "N"
	SoDSecurityTypeBondCollateralizedDebtObligator       SoDSecurityType = "O"
	SoDSecurityTypeBondTreasury                          SoDSecurityType = "P"
	SoDSecurityTypeBondTreasuryNote                      SoDSecurityType = "Q"
	SoDSecurityTypeBondTreasuryBill                      SoDSecurityType = "R"
	SoDSecurityTypeBondConvertibleCorporate              SoDSecurityType = "S"
	SoDSecurityTypeEquityConvertiblePreferred            SoDSecurityType = "T"
	SoDSecurityTypeEquityAlternativeInvestment           SoDSecurityType = "W"
)

func (t SoDSecurityType) Supported() bool {
	switch t {
	case SoDSecurityTypeEquityCommonStock:
		fallthrough
	case SoDSecurityTypeEquityPreferredStock:
		fallthrough
	case SoDSecurityTypeEquityMutualFund:
		fallthrough
	case SoDSecurityTypeEquityMiscStock:
		return true
	default:
		return false
	}
}

type SoDPosition struct {
	AccountNumber          string           `gorm:"type:varchar(13);index"`
	ProcessDate            *string          `sql:"type:date"`
	Firm                   string           `sql:"type:text"`
	CorrespondentID        string           `sql:"type:text"`
	CorrespondentOfficeID  string           `sql:"type:text"`
	OfficeCode             string           `sql:"type:text"`
	RegisteredRepCode      string           `sql:"type:text"`
	AccType                SoDAccountType   `sql:"type:text"`
	Symbol                 string           `sql:"type:text"`
	CUSIP                  string           `sql:"type:text"`
	TradeQuantity          *decimal.Decimal `gorm:"type:decimal"`
	SettleQuantity         *decimal.Decimal `gorm:"type:decimal"`
	CurrencyCode           string           `sql:"type:text"`
	SecurityTypeCode       SoDSecurityType  `sql:"type:text"`
	Description            string           `sql:"type:text"`
	MarginEligibleCode     string           `sql:"type:text"`
	ClosingPrice           *decimal.Decimal `gorm:"type:decimal"`
	LastActivityDate       string           `csv:"skip" sql:"-"`
	LocLocation            string           `csv:"skip" sql:"-"`
	LocMemo                string           `csv:"skip" sql:"-"`
	OptionAmount           *decimal.Decimal `gorm:"type:decimal"`
	ConversionFactor       *decimal.Decimal `gorm:"type:decimal"`
	UnderlyingCusip        string           `sql:"type:text"`
	OptionsSymbolRoot      string           `sql:"type:text"`
	OptionContractDate     *string          `sql:"type:date"`
	StrikePrice            *decimal.Decimal `gorm:"type:decimal"`
	CallPut                string           `sql:"type:text"`
	ExpirationDeliveryDate *string          `sql:"type:date"`
}

type PositionReport struct {
	positions []SoDPosition
}

func (pr *PositionReport) ExtCode() string {
	return "EXT871"
}

func (pr *PositionReport) Delimiter() string {
	return ","
}

func (pr *PositionReport) Header() bool {
	return false
}

func (pr *PositionReport) Extension() string {
	return "CSV"
}

func (pr *PositionReport) Value() reflect.Value {
	return reflect.ValueOf(pr.positions)
}

func (pr *PositionReport) Append(v interface{}) {
	pr.positions = append(pr.positions, v.(SoDPosition))
}

// Sync compares the start of day position records Apex has
// with the records in the DB. It does so by comparing the
// asset, as well as the trade quantities
func (pr *PositionReport) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}

	asOfDate, err := tradingdate.NewFromDate(asOf.Year(), asOf.Month(), asOf.Day())
	if err != nil {
		log.Panic("start of day running for none trading day")
	}

	for _, sodPos := range pr.positions {

		if IsFirmAccount(sodPos.AccountNumber) || !sodPos.SecurityTypeCode.Supported() {
			continue
		}

		acct := &models.Account{}

		tx := db.RepeatableRead()

		// find the account
		q := tx.Where("apex_account = ?", sodPos.AccountNumber).Find(&acct)
		if q.RecordNotFound() {
			tx.Rollback()
			if utils.Prod() {
				errors = append(errors, pr.genError(asOf, sodPos, fmt.Errorf("account not found")))
			}
			continue
		}

		if q.Error != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", pr.ExtCode(), "error", q.Error)
		}

		asset := &models.Asset{}

		// find the asset
		// Apex sends positions SoD sometimes with pre-update CUSIP, so it's important
		// to check both
		q = tx.Where("cusip = ? OR cusip_old = ?", sodPos.CUSIP, sodPos.CUSIP).Find(&asset)

		if q.RecordNotFound() {
			tx.Rollback()
			errors = append(errors, pr.genError(asOf, sodPos, fmt.Errorf("asset not found")))
			continue
		}

		if q.Error != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", pr.ExtCode(), "error", q.Error)
		}

		rawPositions := []*models.Position{}

		// find the open positions at the end of trading date.
		q = tx.
			Where(
				"asset_id = ? AND account_id = ? AND status != ?",
				asset.ID, acct.ID, models.Split).
			Where(
				"entry_timestamp <= ? AND (exit_timestamp > ? OR exit_timestamp IS NULL)",
				asOfDate.SessionClose(), asOfDate.SessionClose()).
			Find(&rawPositions)

		if q.RecordNotFound() {
			tx.Rollback()
			errors = append(errors, pr.genError(asOf, sodPos, fmt.Errorf("position not found")))
			continue
		}

		if q.Error != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", pr.ExtCode(), "error", q.Error)
		}

		if markedForSplit(rawPositions, asOf) {
			var err error

			rawPositions, err = handleSplit(rawPositions, *sodPos.TradeQuantity)
			if err != nil {
				tx.Rollback()
				errors = append(errors, pr.genError(asOf, sodPos, err))
				continue
			}

			for i := range rawPositions {
				if err := tx.Save(rawPositions[i]).Error; err != nil {
					tx.Rollback()
					log.Panic("start of day database error", "file", pr.ExtCode(), "error", q.Error)
				}
			}
		}

		// validate the stored position
		if err := pr.compareQty(totalQty(rawPositions), sodPos); err != nil {
			tx.Rollback()
			errors = append(errors, pr.genError(asOf, sodPos, err))
			continue
		}

		tx.Commit()
	}

	StoreErrors(errors)

	return uint(len(pr.positions) - len(errors)), uint(len(errors))
}

func (pr *PositionReport) genError(asOf time.Time, sodPos SoDPosition, err error) models.BatchError {
	log.Error("start of day error", "file", pr.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":        err.Error(),
		"sod_position": sodPos})
	return models.BatchError{
		ProcessDate:               asOf.Format("2006-01-02"),
		FileCode:                  pr.ExtCode(),
		PrimaryRecordIdentifier:   sodPos.AccountNumber,
		SecondaryRecordIdentifier: sodPos.Symbol,
		Error:                     buf,
	}
}

func (pr *PositionReport) compareQty(totalQty decimal.Decimal, sodPos SoDPosition) error {
	if sodPos.TradeQuantity == nil {
		return fmt.Errorf("trade quantity is nil")
	}
	if !totalQty.Equal(*sodPos.TradeQuantity) {
		return fmt.Errorf(
			"mismatched trade quantities (%s != %s)",
			totalQty.String(),
			sodPos.TradeQuantity.String())
	}
	return nil
}

func totalQty(rawPositions []*models.Position) (qty decimal.Decimal) {
	for _, pos := range rawPositions {
		qty = qty.Add(pos.Qty)
	}
	return
}

func markedForSplit(rawPositions []*models.Position, asOf time.Time) bool {
	asOfDate := asOf.Format("2006-01-02")
	for _, pos := range rawPositions {
		if pos.MarkedForSplitAt != nil {
			if strings.EqualFold(asOfDate, *pos.MarkedForSplitAt) {
				return true
			}
		}
	}
	return false
}

func handleSplit(rawPositions []*models.Position, splitQty decimal.Decimal) (splitPositions []*models.Position, err error) {
	var (
		totalQty = totalQty(rawPositions)
		basis    = costBasis(rawPositions)
	)

	switch {
	// reverse split
	case splitQty.LessThan(totalQty):
		splitPositions, err = splitReverse(rawPositions, totalQty, splitQty)
		if err != nil {
			return
		}
	// normal split
	case splitQty.GreaterThan(totalQty):
		splitPositions, err = splitNormal(rawPositions, totalQty, splitQty)
		if err != nil {
			return
		}
	// no split
	default:
		return nil, fmt.Errorf("no split (shares equal)")
	}

	adjustEntries(splitPositions, basis, splitQty)

	return
}

// ----------- normal split handling -------------
func splitNormal(rawPositions []*models.Position, totalQty, splitQty decimal.Decimal) ([]*models.Position, error) {
	var (
		numRaw      = decimal.Zero
		sharesToAdd = splitQty.Sub(totalQty)
	)

	for _, p := range rawPositions {
		if p.Status != models.Closed {
			numRaw = numRaw.Add(decimal.New(1, 0))
		}
	}

	sharesPerPosition := sharesToAdd.Div(numRaw).Floor()

	for i := range rawPositions {
		oldQty := rawPositions[i].Qty
		rawPositions[i].Qty = oldQty.Add(sharesPerPosition)
		sharesToAdd = sharesToAdd.Sub(sharesPerPosition)
	}

	if sharesToAdd.GreaterThan(decimal.Zero) {
		return splitNormal(rawPositions, totalQty, sharesToAdd.Sub(totalQty))
	}

	for i := range rawPositions {
		rawPositions[i].MarkedForSplitAt = nil
	}

	return rawPositions, nil
}

// ----------- reverse split handling ------------

func splitReverse(rawPositions []*models.Position, totalQty, splitQty decimal.Decimal) ([]*models.Position, error) {
	var (
		numRaw         = decimal.Zero
		sharesToRemove = totalQty.Sub(splitQty)
	)

	for _, p := range rawPositions {
		if p.Status != models.Closed {
			numRaw = numRaw.Add(decimal.New(1, 0))
		}
	}

	sharesPerPosition := sharesToRemove.Div(numRaw).Floor()

	if sharesPerPosition.LessThanOrEqual(decimal.Zero) {
		sharesPerPosition = decimal.New(1, 0)
	}

	for i := range rawPositions {
		if sharesToRemove.LessThanOrEqual(decimal.Zero) {
			break
		}

		if rawPositions[i].Status == models.Closed {
			continue
		}

		closed, removed, err := removeShares(rawPositions[i], sharesPerPosition)
		if err != nil {
			return nil, err
		}

		if closed {
			sharesToRemove = sharesToRemove.Sub(removed)
		} else {
			sharesToRemove = sharesToRemove.Sub(sharesPerPosition)
		}
	}

	if sharesToRemove.GreaterThan(decimal.Zero) {
		return splitReverse(rawPositions, sharesToRemove.Add(splitQty), splitQty)
	}

	return rawPositions, nil
}

func removeShares(
	rawPosition *models.Position,
	sharesToRemove decimal.Decimal,
) (
	closed bool,
	removed decimal.Decimal,
	err error) {

	for removed.LessThan(sharesToRemove) {
		closed, err = removeShare(rawPosition)

		if err != nil {
			return
		}

		removed = removed.Add(decimal.New(1, 0))

		if closed {
			return
		}
	}
	return
}

func removeShare(rawPosition *models.Position) (closed bool, err error) {
	if rawPosition.Qty.LessThanOrEqual(decimal.Zero) {
		err = fmt.Errorf("no shares")
		return
	}
	rawPosition.Qty = rawPosition.Qty.Sub(decimal.New(1, 0))

	if rawPosition.Qty.LessThanOrEqual(decimal.Zero) {
		rawPosition.Status = models.Closed
		closed = true
	}
	return
}

func costBasis(rawPositions []*models.Position) (costBasis decimal.Decimal) {
	for _, p := range rawPositions {
		if p.Status != models.Closed {
			costBasis = costBasis.Add(p.Qty.Mul(p.EntryPrice))
		}
	}
	return
}

func adjustEntries(rawPositions []*models.Position, basis, splitQty decimal.Decimal) {
	for i := range rawPositions {
		if rawPositions[i].Status != models.Closed {
			rawPositions[i].EntryPrice = basis.Div(splitQty)
		}
	}
}
