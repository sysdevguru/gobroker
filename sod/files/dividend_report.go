package files

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDDividendReportNumber string

var (
	DividendExDateReport     SoDDividendReportNumber = "077"
	DividendRecordDateReprot SoDDividendReportNumber = "078"
)

type SoDDividendPositionQuantityLongOrShort string

var (
	DividendPositionQuantityLong  SoDDividendPositionQuantityLongOrShort = "L"
	DividendPositionQuantityShort SoDDividendPositionQuantityLongOrShort = "S"
)

type SoDDividend struct {
	ReportDate                  string                                 `sql:"type:date"`
	ReportNumber                SoDDividendReportNumber                `sql:"type:text"`
	ReportName                  string                                 `sql:"type:text"`
	Symbol                      string                                 `sql:"type:text"`
	CUSIP                       string                                 `sql:"type:text"`
	Description1                string                                 `sql:"type:text"`
	Description2                string                                 `sql:"type:text"`
	Description3                string                                 `sql:"type:text"`
	SecurityTypeCode            string                                 `sql:"type:text"`
	CurrencyCode                string                                 `sql:"type:text"`
	ExchangeDate                string                                 `sql:"type:date"`
	RecordDate                  string                                 `sql:"type:date"`
	PayDate                     string                                 `sql:"type:date"`
	DividendRate                decimal.Decimal                        `gorm:"type:decimal"`
	MaturityDate                *string                                `sql:"type:date"`
	CouponDate                  *string                                `sql:"type:date"`
	FirstCouponDate             *string                                `sql:"type:date"`
	CouponRate                  *decimal.Decimal                       `sql:"type:text"`
	IssueDate                   *string                                `sql:"type:date"`
	AccountNumber               string                                 `gorm:"type:varchar(13);index"`
	AccType                     SoDAccountType                         `sql:"type:text"`
	AccountName                 string                                 `sql:"type:text"`
	Location                    string                                 `csv:"skip" sql:"-"`
	Position                    decimal.Decimal                        `sql:"type:text"`
	PositionQuantityLongOrShort SoDDividendPositionQuantityLongOrShort `sql:"type:text"`
	DividendInterest            *decimal.Decimal                       `gorm:"type:decimal"`
	PayInstruction              string                                 `csv:"skip" sql:"-"`
	WithHoldAmount              *decimal.Decimal                       `gorm:"type:decimal"`
	LastActivity                string                                 `csv:"skip" sql:"-"`
	LongTotalPayPosition        string                                 `csv:"skip" sql:"-"`
	CreditTotalPayAmount        string                                 `csv:"skip" sql:"-"`
	ShortTotalPayPosition       string                                 `csv:"skip" sql:"-"`
	DebitTotalPayAmount         string                                 `csv:"skip" sql:"-"`
	Declared                    string                                 `csv:"skip" sql:"-"`
	AccountStatus               string                                 `csv:"skip" sql:"-"`
	RecordType                  string                                 `csv:"skip" sql:"-"`
}

type DividendReport struct {
	dividends []SoDDividend
}

func (dr *DividendReport) ExtCode() string {
	return "EXT922"
}

func (dr *DividendReport) Delimiter() string {
	return ","
}

func (dr *DividendReport) Header() bool {
	return true
}

func (dr *DividendReport) Extension() string {
	return "csv"
}

func (dr *DividendReport) Value() reflect.Value {
	return reflect.ValueOf(dr.dividends)
}

func (dr *DividendReport) Append(v interface{}) {
	dr.dividends = append(dr.dividends, v.(SoDDividend))
}

func (dr *DividendReport) Sync(asOf time.Time) (uint, uint) {
	errs := []models.BatchError{}

	for _, dividend := range dr.dividends {
		tx := db.DB().Begin()

		var acct models.Account
		if err := tx.Set("gorm:query_option", db.ForUpdate).
			Where("apex_account = ?", dividend.AccountNumber).
			Find(&acct).Error; err != nil {
			// if no-prod and record not found, skip the row because UAT env may include invalid accounts.
			switch {
			case gorm.IsRecordNotFoundError(err) && !utils.Prod():
				tx.Rollback()
				continue
			case gorm.IsRecordNotFoundError(err):
				tx.Rollback()
				errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to find account")))
				continue
			default:
				// database is dead or something let it retry.
				tx.Rollback()
				log.Panic("start of day database error", "file", dr.ExtCode(), "error", err)
			}
		}

		var asset models.Asset
		if err := tx.Where("cusip = ?", dividend.CUSIP).Find(&asset).Error; err != nil {
			switch {
			case gorm.IsRecordNotFoundError(err):
				tx.Rollback()
				errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to find asset")))
				continue
			default:
				// database is dead or something let it retry.
				tx.Rollback()
				log.Panic("start of day database error", "file", dr.ExtCode(), "error", err)
			}
		}

		reportDate, err := date.Parse("01/02/2006", dividend.ReportDate)
		if err != nil {
			tx.Rollback()
			errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse report_date")))
			continue
		}

		recordDate, err := date.Parse("01/02/2006", dividend.RecordDate)
		if err != nil {
			tx.Rollback()
			errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse record_date")))
			continue
		}

		exchangeDate, err := date.Parse("01/02/2006", dividend.ExchangeDate)
		if err != nil {
			tx.Rollback()
			errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse exchange_date")))
			continue
		}

		payDate, err := date.Parse("01/02/2006", dividend.PayDate)
		if err != nil {
			tx.Rollback()
			errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse pay_date")))
			continue
		}

		var maturityDate *date.Date
		if dividend.MaturityDate != nil {
			{
				t, err := date.Parse("01/02/2006", *dividend.MaturityDate)
				if err != nil {
					tx.Rollback()
					errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse maturity_date")))
					continue
				}
				maturityDate = &t
			}
		}

		var firstCouponDate *date.Date
		if dividend.FirstCouponDate != nil {
			{
				t, err := date.Parse("01/02/2006", *dividend.FirstCouponDate)
				if err != nil {
					tx.Rollback()
					errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse first_coupon_date")))
					continue
				}
				firstCouponDate = &t
			}
		}

		var issueDate *date.Date
		if dividend.IssueDate != nil {
			{
				t, err := date.Parse("01/02/2006", *dividend.IssueDate)
				if err != nil {
					tx.Rollback()
					errs = append(errs, dr.genError(asOf, dividend, errors.Wrap(err, "failed to parse issue_date")))
					continue
				}
				issueDate = &t
			}
		}

		patch := models.Dividend{
			Symbol:                      dividend.Symbol,
			CUSIP:                       dividend.CUSIP,
			RecordDate:                  recordDate,
			PayDate:                     payDate,
			DividendRate:                dividend.DividendRate,
			MaturityDate:                maturityDate,
			CouponDate:                  dividend.CouponDate,
			FirstCouponDate:             firstCouponDate,
			CouponRate:                  dividend.CouponRate,
			IssueDate:                   issueDate,
			Position:                    dividend.Position,
			PositionQuantityLongOrShort: enum.DividendPositionQuantityLongOrShort(string(dividend.PositionQuantityLongOrShort)),
			DividendInterest:            dividend.DividendInterest,
			WithHoldAmount:              dividend.WithHoldAmount,
			LastReportDate:              reportDate,
		}

		var div models.Dividend
		if err := tx.Where(models.Dividend{AccountID: acct.ID, ExchangeDate: exchangeDate, AssetID: asset.ID}).
			Assign(patch).FirstOrInit(&div).Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", dr.ExtCode(), "error", err)
		}

		if err := tx.Save(&div).Commit().Error; err != nil {
			log.Panic("start of day database error", "file", dr.ExtCode(), "error", err)
		}
	}

	return uint(len(dr.dividends) - len(errs)), uint(len(errs))
}

func (dr *DividendReport) genError(asOf time.Time, dividend SoDDividend, err error) models.BatchError {
	log.Error("start of day error", "file", dr.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":    err.Error(),
		"dividend": dividend,
	})
	return models.BatchError{
		ProcessDate:             asOf.Format("2006-01-02"),
		FileCode:                dr.ExtCode(),
		PrimaryRecordIdentifier: dividend.AccountNumber,
		Error:                   buf,
	}
}
