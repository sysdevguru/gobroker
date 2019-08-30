package models

import (
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type Transfer struct {
	ID                          string                 `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	ApexID                      *string                `json:"apex_id" sql:"type:text"`
	CreatedAt                   time.Time              `json:"created_at"`
	UpdatedAt                   time.Time              `json:"updated_at"`
	DeletedAt                   *time.Time             `json:"deleted_at"`
	AccountID                   string                 `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	Type                        enum.TransferType      `json:"type" gorm:"not null" sql:"type:text;default:'ach'"`
	RelationshipID              *string                `json:"relationship_id"`
	Amount                      decimal.Decimal        `json:"amount" gorm:"type:decimal;not null"`
	EstimatedFundsAvailableDate *string                `json:"est_funds_available_date" sql:"type:date"`
	Status                      enum.TransferStatus    `json:"status" gorm:"type:varchar(16);not null"`
	Direction                   apex.TransferDirection `json:"direction" gorm:"type:varchar(8);not null"`
	BalanceValidated            *bool                  `json:"balance_validated"`
	BatchProcessedAt            *string                `json:"batch_processed_at" sql:"type:date"`
	ExpiresAt                   *time.Time             `json:"expires_at"`
	Relationship                *ACHRelationship       `json:"-" gorm:"ForeignKey:RelationshipID"`
	Reason                      string                 `json:"reason" sql:"type:text"`
	ReasonCode                  string                 `json:"reason_code" sql:"type:text"`
}

func (t *Transfer) BeforeCreate(scope *gorm.Scope) error {
	if t.ID == "" {
		t.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", t.ID)
}

func (t *Transfer) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(t.ID)
	return id
}

func (t *Transfer) AccountIDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(t.AccountID)
	return id
}

func (t *Transfer) Expired() bool {
	return t.ExpiresAt != nil && t.ExpiresAt.Before(clock.Now())
}
