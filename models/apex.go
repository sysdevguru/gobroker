package models

import (
	"time"

	"github.com/alpacahq/apex"
)

type ALEStatus struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	UpdatedAt time.Time `json:"updated_at"`
	Topic     string    `json:"topic" gorm:"not null;unique_index"`
	Watermark uint64    `json:"watermark"`
}

type Snap struct {
	ID                string          `json:"id" gorm:"type:varchar(100);primary_key"`
	AccountID         string          `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	DocumentRequestID string          `json:"document_request_id" gorm:"not null;index" sql:"type:uuid;"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DeletedAt         *time.Time      `json:"deleted_at"`
	MimeType          string          `json:"mime_type" gorm:"type:text"`
	Name              string          `json:"name" gorm:"type:text"`
	ALEConfirmedAt    *time.Time      `json:"ale_confirmed_at"`
	Type              DocumentType    `json:"type" gorm:"-"`
	SubType           string          `json:"sub_type" gorm:"-"`
	Preview           *string         `json:"preview" gorm:"-"`
	DocumentRequest   DocumentRequest `json:"-" gorm:"ForeignKey:DocumentRequestID"`
}

type InvestigationStatus string

const (
	SketchPending       InvestigationStatus = "PENDING"
	SketchIndeterminate InvestigationStatus = "INDETERMINATE"
	SketchRejected      InvestigationStatus = "REJECTED"
	SketchAppealed      InvestigationStatus = "APPEALED"
	SketchAccepted      InvestigationStatus = "ACCEPTED"
)

func (s InvestigationStatus) Closed() bool {
	switch s {
	case SketchRejected:
		fallthrough
	case SketchAccepted:
		return true
	default:
		return false
	}
}

type Investigation struct {
	ID        string              `json:"id" gorm:"type:varchar(100);primary_key"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt *time.Time          `json:"deleted_at"`
	AccountID string              `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	Status    InvestigationStatus `json:"status" gorm:"type:varchar(13);not null"`
}

type HermesFailure struct {
	ID                string            `json:"id" gorm:"primary_key" sql:"type:text"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	Status            apex.HermesStatus `json:"status" sql:"type:text NOT NULL"`
	Email             string            `json:"email" sql:"type:text NOT NULL"`
	CorrespondentCode string            `json:"correspondent_code" sql:"type:text NOT NULL"`
	Owner             Owner             `json:"owner" gorm:"ForeignKey:Email"`
}
