package models

import (
	"encoding/json"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type ACHRelationship struct {
	ID                 string                  `json:"id" gorm:"type:varchar(100);primary_key"`
	ApexID             *string                 `json:"apex_id" sql:"type:text"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
	DeletedAt          *time.Time              `json:"deleted_at"`
	AccountID          string                  `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	Status             enum.RelationshipStatus `json:"status" gorm:"not null"`
	ApprovalMethod     apex.ACHApprovalMethod  `json:"approval_method" gorm:"not null"`
	PlaidAccount       *string                 `json:"-" gorm:"type:varchar(100)"`
	PlaidToken         *string                 `json:"-" gorm:"type:varchar(100)"`
	PlaidItem          *string                 `json:"-" gorm:"type:varchar(100)"`
	PlaidInstitution   *string                 `json:"institution" gorm:"type:varchar(100)"`
	Mask               *string                 `json:"mask"`
	Nickname           *string                 `json:"nickname"`
	HashBankInfo       []byte                  `json:"-" gorm:"type:bytea"`
	ExpiresAt          *time.Time              `json:"expires_at"`
	FailedAttempts     int                     `json:"failed_attempts" gorm:"not null"`
	Reason             string                  `json:"reason" sql:"type:text"`
	MicroDepositID     string                  `json:"micro_deposit_id" sql:"type:text"`
	MicroDepositStatus enum.TransferStatus     `json:"micro_deposit_status" sql:"type:text"`
}

type BankInfo struct {
	Account          string `json:"account"`
	AccountOwnerName string `json:"account_owner_name"`
	RoutingNumber    string `json:"routing_number"`
	AccountType      string `json:"account_type"`
}

func (ach *ACHRelationship) BeforeCreate(scope *gorm.Scope) error {
	if ach.ID == "" {
		ach.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", ach.ID)
}

func (ach *ACHRelationship) AccountIDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(ach.AccountID)
	return id
}

func (ach *ACHRelationship) Expired() bool {
	return ach.ExpiresAt != nil && ach.ExpiresAt.Before(clock.Now())
}

func (ach *ACHRelationship) GetBankInfo() (*BankInfo, error) {
	buf, err := encryption.DecryptWithkey(ach.HashBankInfo, []byte(env.GetVar("BROKER_SECRET")))
	if err != nil {
		return nil, err
	}

	bankInfo := &BankInfo{}

	if err = json.Unmarshal(buf, &bankInfo); err != nil {
		return nil, err
	}

	return bankInfo, nil
}

func (ach *ACHRelationship) SetBankInfo(info BankInfo) error {
	buf, err := json.Marshal(info)
	if err != nil {
		return err
	}

	ach.HashBankInfo, err = encryption.EncryptWithKey(
		buf, []byte(env.GetVar("BROKER_SECRET")))

	return err
}

func (ach *ACHRelationship) ForApex(acct *Account) (*apex.ACHRelationship, error) {
	bankInfo, err := ach.GetBankInfo()
	if err != nil {
		return nil, err
	}

	apexRel := &apex.ACHRelationship{
		Account:              *acct.ApexAccount,
		BankAccount:          bankInfo.Account,
		BankAccountOwnerName: bankInfo.AccountOwnerName,
		BankRoutingNumber:    bankInfo.RoutingNumber,
		BankAccountType:      bankInfo.AccountType,
		ApprovalMethod:       string(ach.ApprovalMethod),
	}

	if ach.Nickname != nil {
		apexRel.Nickname = *ach.Nickname
	}

	return apexRel, nil
}
