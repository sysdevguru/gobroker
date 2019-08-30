package models

import (
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type Asset struct {
	ID        string           `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	CreatedAt time.Time        `json:"-"`
	UpdatedAt time.Time        `json:"-"`
	Class     enum.AssetClass  `json:"asset_class" gorm:"unique_index:idx_asset_exchange_symbol" sql:"type:text"`
	Exchange  string           `json:"exchange" gorm:"unique_index:idx_asset_exchange_symbol" sql:"type:text"`
	Symbol    string           `json:"symbol" gorm:"unique_index:idx_asset_exchange_symbol" sql:"type:text"`
	SymbolOld string           `json:"-" sql:"type:text"`
	CUSIP     string           `json:"-" gorm:"column:cusip" sql:"type:text"`
	CUSIPOld  string           `json:"-" gorm:"column:cusip_old" sql:"type:text"`
	Status    enum.AssetStatus `json:"status" sql:"type:text"`
	Tradable  bool             `json:"tradable"`
	Shortable bool             `json:"-"`
}

func (a *Asset) BeforeCreate(scope *gorm.Scope) error {
	if a.ID == "" {
		a.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", a.ID)
}

func (a *Asset) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(a.ID)
	return id
}

func (a *Asset) Active() bool {
	return a.Status == enum.AssetActive
}
