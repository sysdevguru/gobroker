package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type Execution struct {
	ID                string             `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	DeletedAt         *time.Time         `json:"deleted_at"`
	Account           string             `fix:"440" json:"account" gorm:"not null" sql:"type:text"`
	BrokerExecID      string             `fix:"17" json:"broker_exec_id" gorm:"not null;unique_index:idx_execution_broker_exec_id_transaction_time" sql:"type:text"`
	Type              enum.ExecutionType `fix:"150" json:"type" gorm:"not null" sql:"type:text"`
	Symbol            string             `fix:"55" json:"symbol" gorm:"not null" sql:"type:text"`
	Side              enum.Side          `fix:"54" json:"side" gorm:"not null" sql:"text"`
	OrderID           string             `json:"order_id" gorm:"not null" sql:"type:uuid"`
	OrderType         enum.OrderType     `fix:"40" json:"order_type" gorm:"not null" sql:"type:text"`
	BrokerOrderID     string             `fix:"37" json:"broker_order_id" gorm:"not null" sql:"type:text"`
	OrderStatus       enum.OrderStatus   `fix:"39" json:"order_status" gorm:"not null" sql:"type:text"`
	Price             *decimal.Decimal   `fix:"31" json:"price" gorm:"type:decimal"`
	Qty               *decimal.Decimal   `fix:"32" json:"qty" gorm:"type:decimal"`
	LeavesQty         *decimal.Decimal   `fix:"151" json:"leaves_qty" gorm:"type:decimal"`
	CumQty            *decimal.Decimal   `fix:"14" json:"cum_qty" gorm:"type:decimal"`
	AvgPrice          *decimal.Decimal   `fix:"6" json:"avg_price" gorm:"type:decimal"`
	TransactionTime   time.Time          `fix:"60" json:"transaction_time" gorm:"not null;unique_index:idx_execution_broker_exec_id_transaction_time"`
	PreviousExecID    *string            `fix:"19" json:"previous_exec_id" sql:"type:text"`
	BraggartTimestamp *time.Time         `json:"braggart_timestamp"`
	BraggartID        *string            `json:"braggart_id" sql:"type:text"`
	BraggartStatus    *string            `json:"braggart_status" sql:"type:varchar(24)"`
	FeeSec            *decimal.Decimal   `json:"fee_sec" gorm:"type:decimal"`
	FeeMisc           *decimal.Decimal   `json:"fee_misc" gorm:"type:decimal"`
	Fee1              *decimal.Decimal   `json:"fee1" gorm:"type:decimal"`
	Fee2              *decimal.Decimal   `json:"fee2" gorm:"type:decimal"`
	Fee3              *decimal.Decimal   `json:"fee3" gorm:"type:decimal"`
	Fee4              *decimal.Decimal   `json:"fee4" gorm:"type:decimal"`
	Fee5              *decimal.Decimal   `json:"fee5" gorm:"type:decimal"`
}

func (e *Execution) BeforeCreate(scope *gorm.Scope) error {
	if e.ID == "" {
		e.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", e.ID)
}

func (e *Execution) HasFee() bool {
	fees := []*decimal.Decimal{
		e.FeeSec, e.FeeMisc, e.Fee1, e.Fee2, e.Fee3, e.Fee4, e.Fee5,
	}
	for i := range fees {
		if fees[i] != nil {
			return true
		}
	}
	return false
}

func (e *Execution) TotalFee() decimal.Decimal {
	fee := decimal.Zero

	fees := []*decimal.Decimal{
		e.FeeSec, e.FeeMisc, e.Fee1, e.Fee2, e.Fee3, e.Fee4, e.Fee5,
	}
	for i := range fees {
		if fees[i] != nil {
			fee = fee.Add(*fees[i])
		}
	}
	return fee
}

func (e *Execution) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(e.ID)
	return id
}

func (e *Execution) OrderIDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(e.OrderID)
	return id
}

func (e *Execution) CostBasis() decimal.Decimal {
	mul := decimal.NewFromFloat(1)
	if e.Side == enum.Sell {
		mul = decimal.NewFromFloat(-1)
	}
	return e.Qty.Mul(*e.Price).Mul(mul)
}

func (e *Execution) Braggart() (*apex.BraggartTransaction, error) {
	tx := &apex.BraggartTransaction{Version: 1, ExternalID: e.ID}
	tx.Transaction.Type = apex.Execution
	tx.Transaction.AccountNumber = e.Account
	// TODO: verify
	tx.Transaction.AccountType = "MARGIN"
	switch e.Side {
	case enum.Buy:
		tx.Transaction.Side.Type = apex.Buy
	case enum.Sell:
		tx.Transaction.Side.Type = apex.Sell
		// case enum.SellShort:
		// 	tx.Transaction.Side.Type = apex.Sell
		// 	tx.Transaction.Side.ShortType = "SHORT"
	}
	if e.Qty == nil {
		return nil, fmt.Errorf("No quantity for execution: %v", e.ID)
	}
	tx.Transaction.Quantity = int(e.Qty.IntPart())
	if e.Price == nil {
		return nil, fmt.Errorf("No price for execution: %v", e.ID)
	}
	tx.Transaction.Price, _ = e.Price.Float64()
	tx.Transaction.Currency = "USD"
	tx.Transaction.TransactionDateTime = e.TransactionTime.In(calendar.NY).Format("2006-01-02T15:04:05.000-07:00")
	tx.Transaction.BrokerCapacity = "AGENT"
	tx.Transaction.Route.Type = "MNGD"
	tx.Transaction.OrderID = e.OrderID
	tx.Instrument.Type = "EQUITY"
	tx.Instrument.InstrumentID.ID = ApexFormat(e.Symbol)
	tx.Instrument.InstrumentID.Type = "TICKER_SYMBOL"
	return tx, nil
}

type FailureReason string

const (
	MarshalFailure  FailureReason = "MARSHAL"
	ServiceFailure  FailureReason = "SERVICE"
	DatabaseFailure FailureReason = "DATABASE"
	RMQFailure      FailureReason = "RMQ"
	FIXFailure      FailureReason = "FIX"
	BustedTrade     FailureReason = "BUSTED_TRADE" // A trade is busted when the exchange corrects a prior execution
)

type TradeFailure struct {
	gorm.Model
	Queue       string          `gorm:"not null" sql:"type:text"`
	Body        json.RawMessage `gorm:"not null" sql:"type:json;"`
	Reason      FailureReason   `gorm:"not null" sql:"type:text"`
	Error       string          `gorm:"not null" sql:"type:text"`
	AccountID   *string         `sql:"type:text;"` // nullable because sometimes may not be available
	ApexAccount *string         `sql:"type:text;"` // nullable because sometimes may not be available
	OrderID     *string         `sql:"type:text;"` // nullable because sometimes may not be available
	RecoveredAt *time.Time
}

// ApexFormat returns the symbol in the format Apex uses
func ApexFormat(symbol string) string {
	// handle preferred shares
	apexSym := strings.Replace(symbol, "-", "PR", 1)

	// handle . symbols
	return strings.Replace(apexSym, ".", "", 1)
}

// FinraFormat returns the symbol in the format FINRA uses
func FinraFormat(symbol string) string {
	// handle preferred shares
	finraSym := strings.Replace(symbol, "-", " PR", 1)

	// handle . symbols
	return strings.Replace(finraSym, ".", " ", 1)
}
