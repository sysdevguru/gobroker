package files

import (
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/external/finra"
	"github.com/alpacahq/gobroker/external/polygon"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SecurityMarket string

func (s SecurityMarket) Supported() bool {
	switch s {
	// pink sheets
	case OTC:
		return false
	// the rest
	default:
		return true
	}
}

const (
	NYSE     SecurityMarket = "1"
	AMEX     SecurityMarket = "2"
	Unknown  SecurityMarket = "F"
	NYSEARCA SecurityMarket = "V"
	OTC      SecurityMarket = "O"
	NMS      SecurityMarket = "Q"
)

type SoDSecurityMaster struct {
	CUSIP               string           `sql:"type:text"`
	MarketSymbol        string           `sql:"type:text"`
	ShortDescription    string           `sql:"type:text"`
	SecurityTypeCode    string           `sql:"type:text"`
	CmpQualCode         string           `sql:"type:text"`
	SecQualCode         string           `sql:"type:text"`
	MarginEligibleCode  string           `sql:"type:text"`
	Margin100ReqCode    string           `sql:"type:text"`
	ForeignCode         string           `sql:"type:text"`
	TradeNumberIntCode  string           `csv:"skip" sql:"-"`
	GsccStlCode         string           `csv:"skip" sql:"-"`
	DTCEligibleCode     string           `sql:"type:text"`
	ListedMarketCode    SecurityMarket   `sql:"type:text"`
	WhenIssuedCode      string           `csv:"skip" sql:"-"`
	BookEntryCode       string           `sql:"type:text"`
	STATE               string           `csv:"skip" sql:"-"`
	ReOrgCode           string           `sql:"type:text"`
	DefaultCode         string           `csv:"skip" sql:"-"`
	MarketMakerCode     string           `csv:"skip" sql:"-"`
	ResearchCode        string           `csv:"skip" sql:"-"`
	NASDTypeCode        string           `csv:"skip" sql:"-"`
	FundsSettlementCode string           `csv:"skip" sql:"-"`
	EuroClearCode       string           `csv:"skip" sql:"-"`
	UserCode1           string           `csv:"skip" sql:"-"`
	UserCode2           string           `csv:"skip" sql:"-"`
	UserCode3           string           `csv:"skip" sql:"-"`
	UserCode4           string           `csv:"skip" sql:"-"`
	UserCode5           string           `csv:"skip" sql:"-"`
	Product             string           `sql:"type:text"`
	TaxCode             string           `sql:"type:text"`
	PreviousPrice       *decimal.Decimal `gorm:"type:decimal"`
	ClosingPrice        *decimal.Decimal `gorm:"type:decimal"`
	AlternateTransfer   string           `csv:"skip" sql:"-"`
	TransferAgent       string           `csv:"skip" sql:"-"`
	Description1        string           `sql:"type:text"`
	Description2        string           `sql:"type:text"`
	Description3        string           `sql:"type:text"`
	LastChangeDate      *string          `sql:"type:date"`
	LastTradeDate       *string          `sql:"type:date"`
	LastPriceDate       *string          `sql:"type:date"`
	SICCode             string           `sql:"type:text"`
	RegTRate            decimal.Decimal  `csv:"skip" sql:"-"`
	MaintenanceRate     decimal.Decimal  `csv:"skip" sql:"-"`
	Maintenance100Price int              `csv:"skip" sql:"-"`
	Fed100Price         int              `csv:"skip" sql:"-"`
	ForeignCountry      string           `sql:"type:text"`
	CMORemicIndicator   string           `sql:"type:text"`
	OldSecurityNumber   string           `sql:"type:text"`
	Label               string           `csv:"skip" sql:"-"`
	ConversionFactor    *decimal.Decimal `gorm:"type:decimal"`
	PayingAgent         string           `csv:"skip" sql:"-"`
	Insured             string           `csv:"skip" sql:"-"`
	MBSCC_SBO_TFT       string           `csv:"skip" sql:"-"`
	PriceTypeCode       string           `csv:"skip" sql:"-"`
	PTCEligibleCode     string           `csv:"skip" sql:"-"`
	ShellCUSIPCode      string           `csv:"skip" sql:"-"`
	ExtMLPCode          string           `csv:"skip" sql:"-"`
	IssueCode           string           `csv:"skip" sql:"-"`
	ElectronicFeedCode  string           `csv:"skip" sql:"-"`
	GSCCEligibleCode    string           `csv:"skip" sql:"-"`
	Routing             string           `sql:"type:text"`
	UnderlyingCUSIP     string           `sql:"type:text"`
	StrikePrice         *decimal.Decimal `gorm:"type:decimal"`
	ExpireDate          *string          `sql:"type:date"`
	ProcessDate         *string          `sql:"type:date"`
}

func (s *SoDSecurityMaster) isValid() bool {
	return (strings.EqualFold(s.SecurityTypeCode, SecurityTypeStock) ||
		strings.EqualFold(s.SecurityTypeCode, SecurityTypeFund)) &&
		s.ListedMarketCode != "" &&
		s.Routing != "" &&
		strings.EqualFold(s.ForeignCountry, "US")
}

type SecurityMaster struct {
	securities []SoDSecurityMaster
}

func (sm *SecurityMaster) ExtCode() string {
	return "EXT747"
}

func (sm *SecurityMaster) Delimiter() string {
	return "|"
}

func (sm *SecurityMaster) Header() bool {
	return false
}

func (sm *SecurityMaster) Extension() string {
	return "txt"
}

func (sm *SecurityMaster) Value() reflect.Value {
	return reflect.ValueOf(sm.securities)
}

func (sm *SecurityMaster) Append(v interface{}) {
	sm.securities = append(sm.securities, v.(SoDSecurityMaster))
}

// Sync updates the assets and fundamentals tables of gobroker
// using not only the security_master start of day file, but
// also the intersection of polygon and FINRA symbol lists
// to ensure that all of the assets in the DB are fully supported
// both from Apex's perspective, but also from our data sources.
func (sm *SecurityMaster) Sync(asof time.Time) (uint, uint) {
	// gather the on disk assets
	assets := sm.gatherAssets()

	// gather the securities from the file
	secs := sm.gatherApexSecurities()

	// gather polygon symbols
	poly := sm.gatherPolygonSecurities()
	if poly == nil {
		log.Panic("failed to gather polygon securities")
	}

	// apply finra exchange to polygon securities

	log.Debug("gathered asset data",
		"polygon", len(*poly),
		"sec_master", len(secs),
		"assets", len(assets))

	// Here we are going to range over the assets in the DB.
	// If there are any assets that have a cusip that is not
	// in the latest security master, or has been moved to an
	// unsupported market, or has a symbol that is not validated
	// by polygon's symbol list, then we will mark it inactive.
	// Valid symbols get removed from the security_master
	// hash-map, and any remaining are presumed to be new symbols
	// that must be added to the DB, and will be handled after
	// the removals.
	for _, asset := range assets {
		_, ok := secs[asset.CUSIP]
		polySec, valid := poly.SymbolValid(asset.Symbol)

		if ok && valid {
			// make sure the exchange is up-to-date
			if !strings.EqualFold(asset.Exchange, *polySec.exchange) {
				log.Info(
					"updating asset exchange",
					"symbol", polySec.symbol,
					"previous", asset.Exchange,
					"new", *polySec.exchange)

				updates := map[string]interface{}{"exchange": *polySec.exchange}

				if err := updateAsset(&asset, updates); err != nil {
					log.Panic(
						"start of day database error",
						"file", sm.ExtCode(),
						"error", err)
				}
			}

			if !(asset.Active() && asset.Tradable) {
				log.Info("marking asset tradable", "symbol", asset.Symbol)

				updates := map[string]interface{}{
					"status":   enum.AssetActive,
					"tradable": true,
				}

				if err := updateAsset(&asset, updates); err != nil {
					log.Panic(
						"start of day database error",
						"file", sm.ExtCode(),
						"error", err)
				}
			}
		} else if asset.Status == enum.AssetActive || asset.Tradable {
			log.Info("marking asset untradable", "symbol", asset.Symbol)

			updates := map[string]interface{}{
				"status":   enum.AssetInactive,
				"tradable": false,
			}

			if err := updateAsset(&asset, updates); err != nil {
				log.Panic(
					"start of day database error",
					"file", sm.ExtCode(),
					"error", err)
			}
		}

		// remove the handled symbol from the hash-map
		delete(secs, asset.CUSIP)
	}

	// create the remaining new securities
	// that don't already exist in the DB
	for _, sec := range secs {
		polySec, valid := poly.SymbolValid(sec.MarketSymbol)

		if valid {
			log.Debug("adding security", "symbol", polySec.symbol)

			asset := &models.Asset{
				Class:    enum.AssetClassUSEquity,
				Exchange: *polySec.exchange,
				Symbol:   polySec.symbol,
				CUSIP:    sec.CUSIP,
				Status:   enum.AssetActive,
				Tradable: true,
			}

			tx := db.Begin()

			if err := tx.Where(&models.Asset{
				Class:    asset.Class,
				Exchange: asset.Exchange,
				Symbol:   polySec.symbol,
			}).Assign(&models.Asset{
				CUSIP:    sec.CUSIP,
				Status:   enum.AssetActive,
				Tradable: true}).FirstOrCreate(asset).Error; err != nil {

				tx.Rollback()

				log.Panic(
					"start of day database error",
					"file", sm.ExtCode(),
					"error", err)
			}

			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				log.Panic(
					"start of day database error",
					"file", sm.ExtCode(),
					"error", err)
			}
		} else {
			log.Debug(
				"skipping apex security",
				"symbol", sec.MarketSymbol)
		}
	}

	count := 0

	db.DB().Model(&models.Asset{}).Count(&count)

	return uint(count), 0
}

// gatherAssets returns a hash-map where key is cusip
// and value is the asset stored in the DB
func (sm *SecurityMaster) gatherAssets() map[string]models.Asset {
	assets := []models.Asset{}
	m := map[string]models.Asset{}
	db.DB().Where("class = ?", enum.AssetClassUSEquity).Find(&assets)
	for _, asset := range assets {
		m[asset.CUSIP] = asset
	}
	return m
}

type security struct {
	symbol     string
	exchange   *string
	apexAlias  string
	finraAlias string
}

type polygonSecurities []security

func (ps *polygonSecurities) SymbolValid(symbol string) (*security, bool) {
	if ps == nil {
		return nil, false
	}

	for i := range *ps {
		s := (*ps)[i]

		// exchange was validated by FINRA, and symbol is matched
		if s.exchange != nil && (strings.EqualFold(s.symbol, symbol) ||
			strings.EqualFold(s.apexAlias, symbol)) {
			return &s, true
		}
	}

	return nil, false
}

// gatherPolygon queries symbol list from polygon's API
func (sm *SecurityMaster) gatherPolygonSecurities() *polygonSecurities {

	log.Info("listing polygon symbols")

	start := time.Now()

	resp, err := polygon.ListSymbols()
	if err != nil {
		log.Panic(
			"sod file error",
			"file", sm.ExtCode(),
			"error", err,
		)
	}

	log.Info("listed polygon symbols", "count", len(resp.Symbols), "elapsed", time.Now().Sub(start))

	log.Info("listing FINRA securities")

	start = time.Now()

	finra, err := finra.GetSecurities()
	if err != nil {
		log.Panic("failed to gather FINRA securities")
	}

	log.Info("listed FINRA securities", "count", len(finra), "elapsed", time.Now().Sub(start))

	ps := make(polygonSecurities, len(resp.Symbols))

	for i, symbol := range resp.Symbols {
		s := security{
			symbol:     symbol.Symbol,
			apexAlias:  models.ApexFormat(symbol.Symbol),
			finraAlias: models.FinraFormat(symbol.Symbol),
		}

		for _, sec := range finra {
			if strings.EqualFold(sec.Symbol, s.finraAlias) {
				ex := sec.Exchange
				s.exchange = &ex

			}
		}

		ps[i] = s
	}

	return &ps
}

// supported securities (funds -> ETFs)
const (
	SecurityTypeStock string = "A"
	SecurityTypeFund  string = "C"
)

func (sm *SecurityMaster) gatherApexSecurities() map[string]*SoDSecurityMaster {
	m := map[string]*SoDSecurityMaster{}
	for _, sec := range sm.securities {
		if sec.isValid() {
			security := sec
			m[sec.CUSIP] = &security
		}
	}
	return m
}

func updateAsset(asset *models.Asset, updates map[string]interface{}) error {
	tx := db.Begin()

	if err := tx.Model(asset).Update(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
