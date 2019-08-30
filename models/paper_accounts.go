package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

// PaperAccount is the alias of paper trading accout.
type PaperAccount struct {
	ID             uuid.UUID `json:"-" sql:"type:uuid;"`
	AccountID      uuid.UUID `json:"account_id" sql:"type:uuid;"`
	PaperAccountID uuid.UUID `json:"paper_account_id" sql:"type:uuid;unique_index"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Account Account `json:"-" gorm:"ForeignKey:AccountID;"`
}

func (a *PaperAccount) BeforeCreate(scope *gorm.Scope) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.Must(uuid.NewV4())
	}
	return scope.SetColumn("id", a.ID.String())
}
