package files

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gopaca/log"
)

type SoDReturnedMail struct {
	Firm                string  `sql:"type:text"`
	AccountNumber       string  `gorm:"type:varchar(13);index"`
	MailTypeID          string  `sql:"type:text"`
	MailTypeDescription string  `sql:"type:text"`
	ProcessDate         *string `sql:"type:date"`
	CorrespondentCode   string  `sql:"type:text"`
	OfficeCode          string  `sql:"type:text"`
	AccountName         string  `sql:"type:text"`
	AddressLine1        string  `sql:"type:text"`
	AddressLine2        string  `sql:"type:text"`
	AddressLine3        string  `sql:"type:text"`
	AddressLine4        string  `sql:"type:text"`
	City                string  `sql:"type:text"`
	State               string  `sql:"type:text"`
	ZipCode             string  `sql:"type:text"`
}

type ReturnedMailReport struct {
	returned []SoDReturnedMail
}

func (rmr *ReturnedMailReport) ExtCode() string {
	return "EXT986"
}

func (rmr *ReturnedMailReport) Delimiter() string {
	return ","
}

func (rmr *ReturnedMailReport) Header() bool {
	return true
}

func (rmr *ReturnedMailReport) Extension() string {
	return "CSV"
}

func (rmr *ReturnedMailReport) Value() reflect.Value {
	return reflect.ValueOf(rmr.returned)
}

func (rmr *ReturnedMailReport) Append(v interface{}) {
	rmr.returned = append(rmr.returned, v.(SoDReturnedMail))
}

// Sync goes through the returned mail reports, and reports them
// internally through slack to the designated mail failure channel
func (rmr *ReturnedMailReport) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}

	for _, ret := range rmr.returned {

		if IsFirmAccount(ret.AccountNumber) {
			continue
		}

		msg := slack.NewMailFailure()
		msg.SetBody(map[string]interface{}{
			"account":   ret.AccountNumber,
			"mail_type": ret.MailTypeDescription,
		})

		slack.Notify(msg)
	}

	StoreErrors(errors)

	return uint(len(rmr.returned) - len(errors)), uint(len(errors))
}

func (rmr *ReturnedMailReport) genError(asOf time.Time, ret SoDReturnedMail, err error) models.BatchError {
	log.Error("start of day error", "file", rmr.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":         err.Error(),
		"returned_mail": ret,
	})
	return models.BatchError{
		ProcessDate:             asOf.Format("2006-01-02"),
		FileCode:                rmr.ExtCode(),
		PrimaryRecordIdentifier: ret.AccountNumber,
		Error:                   buf,
	}
}
