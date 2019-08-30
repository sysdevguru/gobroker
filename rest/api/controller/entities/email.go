package entities

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gopaca/clock"
	"github.com/gofrs/uuid"
)

type EmailRequest struct {
	Type      mailer.MailType `json:"type"`
	AccountID string          `json:"account_id"`
	DeliverAt *time.Time      `json:"deliver_at"`
}

func (req *EmailRequest) Validate() (*uuid.UUID, error) {
	acctUUID, err := uuid.FromString(req.AccountID)
	if err != nil {
		return nil, gberrors.InvalidRequestParam.WithMsg(fmt.Sprintf("account_id is invalid (%s)", err))
	}
	if req.DeliverAt != nil && req.DeliverAt.Before(clock.Now()) {
		return nil, gberrors.InvalidRequestParam.WithMsg("deliver_at is invalid")
	}
	return &acctUUID, nil
}
