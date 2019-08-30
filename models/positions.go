package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

type PositionStatus string

const (
	Open   PositionStatus = "open"
	Closed PositionStatus = "closed"
	Split  PositionStatus = "split"
)

type PositionSide string

const (
	Long  PositionSide = "long"
	Short PositionSide = "short"
)

type Position struct {
	ID                 uint             `json:"id" gorm:"primary_key"`
	AssetID            uuid.UUID        `json:"asset_id" gorm:"not null;" sql:"type:uuid;"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	DeletedAt          *time.Time       `json:"deleted_at"`
	AccountID          string           `json:"account_id" gorm:"not null;index"  sql:"type:uuid;"`
	Status             PositionStatus   `json:"status" gorm:"not null;index:idx_position_status;type:varchar(6)"`
	EntryOrderID       string           `json:"entry_order_id" gorm:"not null;index" sql:"type:uuid"`
	ExitOrderID        *string          `json:"exit_order_id" sql:"type:uuid"`
	OriginalPositionID *uint            `json:"original_position_id" gorm:"type:integer"`
	Side               PositionSide     `json:"side" gorm:"not null;type:varchar(5)"`
	Qty                decimal.Decimal  `json:"qty" gorm:"type:decimal;not null;"`
	EntryPrice         decimal.Decimal  `json:"entry_price" gorm:"type:decimal;not null;"`
	EntryTimestamp     time.Time        `json:"entry_timestamp" gorm:"type:timestamp with time zone;not null;"`
	ExitPrice          *decimal.Decimal `json:"exit_price" gorm:"type:decimal"`
	ExitTimestamp      *time.Time       `json:"exit_timestamp" gorm:"type:timestamp with time zone"`
	GrossProfitLoss    *decimal.Decimal `json:"gross_profit_loss" gorm:"type:decimal"`
	MarkedForSplitAt   *string          `json:"-" sql:"type:date"`
	Asset              Asset            `json:"-" gorm:"ForeignKey:AssetID"`
}

// Day's profit loss snapshot
type DayPLSnapshot struct {
	ID         uint            `json:"id" gorm:"primary_key"`
	CreatedAt  time.Time       `json:"created_at"`
	AccountID  string          `json:"account_id" gorm:"not null;unique_index:day_pl_snapshots_unique" sql:"type:uuid;"`
	ProfitLoss decimal.Decimal `json:"profit_loss" gorm:"not null;type:decimal"`
	Basis      decimal.Decimal `json:"basis" gorm:"not null;type:decimal"`
	Date       string          `json:"date" gorm:"not null;unique_index:day_pl_snapshots_unique" sql:"type:date"`
}

func (s DayPLSnapshot) DateString() string {
	return s.Date[:10]
}

func (DayPLSnapshot) TableName() string {
	return "day_pl_snapshots"
}
