package models

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/price"
	"github.com/alpacahq/polycache/rest/client"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type Order struct {
	ID                 string              `fix:"37" json:"id" gorm:"primary_key" sql:"type:uuid;"`
	ClientOrderID      string              `fix:"11" json:"client_order_id" gorm:"not null;unique_index:idx_client_order_id_account" sql:"type:varchar(50);"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	DeletedAt          *time.Time          `json:"deleted_at"`
	SubmittedAt        time.Time           `json:"submitted_at" gorm:"index"`
	FilledAt           *time.Time          `json:"filled_at"`
	ExpiredAt          *time.Time          `json:"expired_at"`
	CanceledAt         *time.Time          `json:"canceled_at"`
	CancelRequestedAt  *time.Time          `json:"cancel_requested_at"`
	ReplacedAt         *time.Time          `json:"replaced_at"`
	ReplaceRequestedAt *time.Time          `json:"replace_requested_at"`
	ReplacedBy         *string             `json:"replaced_by" sql:"type:uuid;"`
	Replaces           *string             `json:"replaces" sql:"type:uuid;"`
	FailedAt           *time.Time          `json:"failed_at"`
	Account            string              `fix:"440" json:"account" gorm:"not null;index;unique_index:idx_client_order_id_account" sql:"type:text"`
	OrderCapacity      enum.OrderCapacity  `fix:"47" json:"order_capacity" gorm:"not null" sql:"type:text;default:'agency'"`
	Qty                decimal.Decimal     `fix:"38" json:"qty" gorm:"type:decimal; not null"`
	AssetID            string              `json:"asset_id" gorm:"not null" sql:"type:text"`
	Symbol             string              `fix:"55" json:"symbol" gorm:"not null" sql:"type:text"`
	SymbolSuffix       string              `fix:"65" json:"symbol_suffix" sql:"type:text"`
	Type               enum.OrderType      `fix:"40" json:"order_type" gorm:"not null" sql:"type:text"`
	ClientOrderType    enum.OrderType      `json:"-" gorm:"not null" sql:"type:text"`
	LimitPrice         *decimal.Decimal    `fix:"44" json:"limit_price" gorm:"type:decimal"`
	Side               enum.Side           `fix:"54" json:"side" gorm:"not null" sql:"type:text"`
	TimeInForce        enum.TimeInForce    `fix:"59" json:"time_in_force" gorm:"not null" sql:"type:text"`
	StopPrice          *decimal.Decimal    `fix:"99" json:"stop_price" gorm:"type:decimal"`
	ExecInst           enum.ExecInst       `fix:"18" json:"exec_inst" gorm:"not null" sql:"type:text;default:'held'"`
	SettlementType     enum.SettlementType `fix:"63" json:"settlement_type" sql:"type:text;default:'regular'"`
	HandlInst          enum.HandlInst      `fix:"21" json:"handl_inst" gorm:"not null" sql:"type:text;default:'manual'"`
	SecurityType       enum.SecurityType   `fix:"167" json:"security_type" gorm:"not null" sql:"type:text;default:'common_stock'"`
	Status             enum.OrderStatus    `json:"order_status" gorm:"not null;index:idx_order_status" sql:"type:text;default:'new'"`
	FilledQty          *decimal.Decimal    `json:"filled_qty" gorm:"type:decimal"`
	FilledAvgPrice     *decimal.Decimal    `json:"filled_avg_price" gorm:"type:decimal"`
	CancelGUID         *string             `json:"cancel_guid" sql:"type:text"`
	TraderInitials     string              `fix:"116" json:"trader_initials" gorm:"type:text"`
	Fee                *decimal.Decimal    `json:"fee" gorm:"type:decimal"` // Only for sell orders we expected to have fee.
	IsCorrection       bool                `json:"-" gorm:"not null" sql:"default:'FALSE'"`
	// Relations
	Executions    []Execution    `json:"-" gorm:"ForeignKey:OrderID"`
	TradeFailures []TradeFailure `json:"-" gorm:"ForeignKey:OrderID"`
}

func (o *Order) BeforeCreate(scope *gorm.Scope) error {
	if o.ID == "" {
		o.ID = uuid.Must(uuid.NewV4()).String()
	}
	if err := scope.SetColumn("id", o.ID); err != nil {
		return err
	}

	if o.ClientOrderID == "" {
		o.ClientOrderID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("client_order_id", o.ClientOrderID)
}

func (o *Order) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(o.ID)
	return id
}

func (o *Order) SetSymbol(symbol string) {
	if parts := strings.Split(symbol, "."); len(parts) > 1 {
		o.Symbol = parts[0]
		o.SymbolSuffix = parts[1]
	} else {
		o.Symbol = symbol
	}
}

func (o *Order) GetSymbol() string {
	symbol := o.Symbol
	if o.SymbolSuffix != "" {
		return strings.Join([]string{symbol, o.SymbolSuffix}, ".")
	}
	return symbol
}

func (o *Order) SetInitials(name string) {
	var buffer bytes.Buffer
	parts := strings.Split(name, " ")
	for _, part := range parts {
		if len(part) > 0 {
			buffer.WriteString(string(part[0]))
		}
	}
	o.TraderInitials = buffer.String()
}

func (o *Order) ForJSON(acctID string) map[string]interface{} {
	return map[string]interface{}{
		"id":              o.ID,
		"client_order_id": o.ClientOrderID,
		"account_id":      acctID,
		"asset_id":        o.AssetID,
	}
}

func (o *Order) Update(e *Execution) *Order {
	// we will take this order status for anything
	// other than rejections or fills. in those cases,
	// we will manually set the status based on the
	// execution type since sometimes they mismatch.
	o.Status = e.OrderStatus

	switch e.Type {
	case enum.ExecutionExpired:
		o.Status = enum.OrderExpired
		o.ExpiredAt = &e.TransactionTime
	case enum.ExecutionRejected:
		o.Status = enum.OrderRejected
		o.FailedAt = &e.TransactionTime
	case enum.ExecutionCanceled:
		o.Status = enum.OrderCanceled
		o.CanceledAt = &e.TransactionTime
	case enum.ExecutionFill:
		o.Status = enum.OrderFilled
		o.FilledAt = &e.TransactionTime
		o.FilledAvgPrice = e.AvgPrice
		o.FilledQty = e.CumQty
	case enum.ExecutionPartialFill:
		o.Status = enum.OrderPartiallyFilled
		o.FilledAt = &e.TransactionTime
		o.FilledAvgPrice = e.AvgPrice
		o.FilledQty = e.CumQty
	case enum.ExecutionPendingNew:
		o.Status = enum.OrderPendingNew
	case enum.ExecutionPendingCancel:
		o.Status = enum.OrderPendingCancel
	case enum.ExecutionPendingReplace:
		o.Status = enum.OrderPendingReplace
	}

	return o
}

func ToLimit(o *Order, buyingPower decimal.Decimal) error {
	if o.Type == enum.Limit || o.Type == enum.StopLimit || o.Side == enum.Sell {
		return nil
	}
	prices, err := client.GetTrades([]string{o.GetSymbol()})
	if err != nil || prices == nil {
		return fmt.Errorf("%v price not found", o.GetSymbol())
	}

	px := decimal.NewFromFloat(prices[o.GetSymbol()].Price)

	limit, _ := price.FormatForOrder(px.Mul(decimal.NewFromFloat(1.05)))
	if o.Qty.GreaterThan(buyingPower.Div(limit)) {
		return errors.New("insufficient buying power")
	}
	o.LimitPrice = &limit
	if o.Type == enum.Market {
		o.Type = enum.Limit
	} else if o.Type == enum.Stop {
		o.Type = enum.StopLimit
	}
	return nil
}

// CostBasis calculates the cost basis of an order. If true
// is passed in, the FilledQty shares will be subtracted
// from the cost basis of the order.
func CostBasis(order *Order, subtractFilled bool) (costBasis decimal.Decimal) {
	filled := decimal.Zero
	if order.FilledQty != nil && subtractFilled {
		filled = *order.FilledQty
	}

	if order.FilledAvgPrice != nil {
		costBasis = order.FilledAvgPrice.Mul(order.Qty.Sub(filled))
	} else {
		costBasis = order.LimitPrice.Mul(order.Qty.Sub(filled))
	}

	if order.Side == enum.Sell {
		costBasis = costBasis.Mul(decimal.NewFromFloat(-1))
	}
	return costBasis
}

func (o *Order) LimitForJSON() *decimal.Decimal {
	if o.ClientOrderType == enum.Market {
		return nil
	}
	return o.LimitPrice
}
