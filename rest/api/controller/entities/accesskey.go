package entities

import (
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/gofrs/uuid"
)

// NOTE: this is a workaround struct for now
// so that we can get the hashed PW and salt
// from papertrader to cache to redis. when
// these codebases are unified, we will no
// longer need this, and this structure
// shouldn't be returned via gobroker's
// API.
type AccessKeyEntity struct {
	ID         string               `json:"id"`
	AccountID  uuid.UUID            `json:"account_id"`
	HashSecret []byte               `json:"hash_secret"`
	Secret     string               `json:"secret"`
	Salt       string               `json:"salt"`
	Status     enum.AccessKeyStatus `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	DeletedAt  *time.Time           `json:"deleted_at"`
}

func (a *AccessKeyEntity) Model() *models.AccessKey {
	return &models.AccessKey{
		ID:         a.ID,
		AccountID:  a.AccountID,
		HashSecret: a.HashSecret,
		Secret:     a.Secret,
		Salt:       a.Salt,
		Status:     a.Status,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
		DeletedAt:  a.DeletedAt,
	}
}
