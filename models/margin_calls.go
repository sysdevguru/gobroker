package models

import (
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/shopspring/decimal"
)

// MarginCall has a composite primary key to uniquely identify each margin call
// and make the start of day file parsing easily idempotent [account:call_type:trade_date].
type MarginCall struct {
	AccountID      string              `json:"account_id" gorm:"primary_key"  sql:"type:uuid;"`
	CallType       enum.MarginCallType `json:"call_type" gorm:"primary_key" sql:"type:text;"`
	CallAmount     decimal.Decimal     `json:"call_amount" sql:"type:decimal not null;"`
	DueDate        string              `json:"due_date" sql:"type:date not null;"`
	TradeDate      string              `json:"trade_date" gorm:"primary_key" sql:"type:date;"`
	LastNotifiedAt *time.Time          `json:"notified_at"`
	MetAt          *time.Time          `json:"met_at"`
}

// ShouldNotify determines whether a customer notification is necessary
// for this account based on whether or not it has been met, and when
// the customer was last notified regarding this specific call.
func (mc *MarginCall) ShouldNotify(deliverAt *time.Time) bool {
	deliveryTime := clock.Now()

	if deliverAt != nil {
		deliveryTime = *deliverAt
	}

	// if the margin call isn't met, and the account owner hasn't
	// been notified yet, or there has been at least 24 hours since
	// the last notification was sent, then we can notify.
	if mc.MetAt == nil && (mc.LastNotifiedAt == nil || deliveryTime.Sub(*mc.LastNotifiedAt) > 24*time.Hour) {
		return true
	}

	return false
}
