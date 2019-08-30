package gbtrade

import (
	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/workers/trade"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

func sendOrderExecutedNotification(tx *gorm.DB, accountID uuid.UUID, execution *models.Execution) error {
	svc := account.Service().WithTx(tx)
	acct, err := svc.GetByID(accountID)
	if err != nil {
		return err
	}
	// Should be sent out msg to queue and send email in different process later.
	go mailer.SendOrderExecutedNotification(
		*acct.ApexAccount,
		*acct.PrimaryOwner().Details.GivenName,
		acct.PrimaryOwner().Email,
		execution, nil)
	return nil
}

// NewTradeWorker returns TradeWorker configured with gobroker specific requirements.
func NewTradeWorker() *trade.TradeWorker {
	return trade.NewTradeWorker(
		db.DB(),
		env.GetVar("EXECUTIONS_QUEUE"),
		env.GetVar("CANCEL_REJECTIONS_QUEUE"),
		"stream",
		"gobroker",
		gbreg.Services,
		sendOrderExecutedNotification,
	)
}
