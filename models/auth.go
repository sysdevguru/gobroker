package models

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

type AccessKey struct {
	ID          string               `json:"id" gorm:"primary_key:true;type:varchar(20);not null;"`
	AccountID   uuid.UUID            `json:"account_id" gorm:"not null" sql:"type:uuid references accounts(id);"`
	HashSecret  []byte               `json:"-" gorm:"type:bytea;not null"`
	Secret      string               `json:"secret" gorm:"-"`
	Salt        string               `json:"-" gorm:"not null;"`
	Status      enum.AccessKeyStatus `json:"status" gorm:"not null"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DeletedAt   *time.Time           `json:"deleted_at"`
	Expiration  *time.Time           `json:"-" gorm:"-"`
	DataSources []string             `json:"-" gorm:"-"`

	Account Account `json:"-" gorm:"ForeignKey:AccountID;"`
}

func (a *AccessKey) Verify(secret string) error {
	hashed, err := encryption.SaltEncrypt([]byte(secret), []byte(a.Salt))

	if err != nil {
		return err
	}

	if bytes.Equal(hashed, a.HashSecret) {
		return nil
	}

	return fmt.Errorf("verification failure")
}

func (a *AccessKey) Expired() bool {
	return a.Expiration != nil && a.Expiration.Before(time.Now())
}

func NewAccessKey(AccountID uuid.UUID, version enum.AccountType) (*AccessKey, error) {
	var prefix string
	switch version {
	case enum.PaperAccount:
		prefix = "PK"
	case enum.CryptoAccount:
		prefix = "CK"
	default:
		prefix = "AK"
	}

	key := AccessKey{
		ID:        fmt.Sprintf("%v%v", prefix, getStringWithCharset(18, charset)),
		Secret:    getStringWithCharset(40, mixedCharset),
		Salt:      getStringWithCharset(4, charset),
		Status:    enum.AccessKeyActive,
		AccountID: AccountID,
	}

	hashed, err := encryption.SaltEncrypt([]byte(key.Secret), []byte(key.Salt))
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt with salt")
	}
	key.HashSecret = hashed

	return &key, nil
}

var seededRand = rand.New(rand.NewSource(clock.Now().UnixNano()))

const (
	charset      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	mixedCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/"
)

func getStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
