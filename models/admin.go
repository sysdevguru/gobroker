package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type Administrator struct {
	ID           string     `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
	Email        string     `valid:"email" json:"email" gorm:"type:varchar(100);unique_index"`
	Name         string     `json:"name" gorm:"type:varchar(100)"`
	HashPassword []byte     `json:"-" gorm:"type:bytea"`
	Salt         []byte     `json:"-" gorm:"type:bytea"`
}

func (a *Administrator) BeforeCreate(scope *gorm.Scope) error {
	if a.ID == "" {
		if id, err := uuid.NewV4(); err != nil {
			return err
		} else {
			return scope.SetColumn("id", id.String())
		}
	}
	return nil
}

func (a *Administrator) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(a.ID)
	return id
}

type AdminNote struct {
	ID        uint           `json:"id" gorm:"primary_key"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"-"`
	Body      string         `json:"body" sql:"type:text;not null;"`
	AccountID string         `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	AdminID   string         `json:"admin_id" gorm:"not null;" sql:"type:uuid;"`
	Admin     *Administrator `json:"-" gorm:"ForeignKey:AdminID"`
}

type AdminEmail struct {
	ID        uint           `json:"id" gorm:"primary_key"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"-"`
	Subject   string         `json:"subject" sql:"type:text;not null;"`
	Body      string         `json:"body" sql:"type:text;not null;"`
	AccountID string         `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	AdminID   string         `json:"admin_id" gorm:"not null;index" sql:"type:uuid;"`
	Admin     *Administrator `json:"-" gorm:"ForeignKey:AdminID"`
}
