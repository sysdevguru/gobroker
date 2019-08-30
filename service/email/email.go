package email

import (
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type EmailService interface {
	Create(acct *models.Account, mailType mailer.MailType, deliverAt *time.Time) error
	WithTx(tx *gorm.DB) EmailService
}

type emailService struct {
	tx             *gorm.DB
	sendMarginCall func(
		acct, givenName, email string,
		dueDate time.Time,
		callAmount decimal.Decimal,
		deliverAt *time.Time) error

	sendPatternDayTrader func(
		acct, givenName, email string,
		dueDate time.Time,
		callAmount decimal.Decimal,
		deliverAt *time.Time) error
}

func Service() EmailService {
	return &emailService{
		sendMarginCall:       mailer.SendMarginCall,
		sendPatternDayTrader: mailer.SendPDTCall,
	}
}

func (s *emailService) WithTx(tx *gorm.DB) EmailService {
	s.tx = tx
	return s
}

func (s *emailService) Create(acct *models.Account, mailType mailer.MailType, deliverAt *time.Time) error {
	owner := acct.PrimaryOwner()
	if err := s.tx.Model(owner).Related(&owner.Details, "Details").Error; err != nil {
		return errors.Wrap(err, "failed to load details")
	}
	switch mailType {
	case mailer.MarginCall:
		if err := s.tx.Model(acct).Related(&acct.MarginCalls, "MarginCalls").Error; err != nil {
			return gberrors.InternalServerError.WithMsg("failed to retrieve margin calls")
		}
		for _, marginCall := range acct.MarginCalls {
			if marginCall.ShouldNotify(deliverAt) {
				dueDate, err := time.Parse(time.RFC3339, marginCall.DueDate)
				if err != nil {
					return err
				}
				if err = s.sendMarginCall(
					*acct.ApexAccount,
					owner.Email,
					*owner.Details.GivenName,
					dueDate,
					marginCall.CallAmount,
					deliverAt); err != nil {
					return err
				}
			}
		}
	case mailer.PatternDayTrader:
		if err := s.tx.Model(acct).Related(&acct.MarginCalls, "MarginCalls").Error; err != nil {
			return gberrors.InternalServerError.WithMsg("failed to retrieve margin calls")
		}
		for _, marginCall := range acct.MarginCalls {
			if marginCall.CallType == enum.EquityMaintenance && marginCall.ShouldNotify(deliverAt) {
				dueDate, err := time.Parse(time.RFC3339, marginCall.DueDate)
				if err != nil {
					return err
				}
				if err = s.sendPatternDayTrader(
					*acct.ApexAccount,
					owner.Email,
					*owner.Details.GivenName,
					dueDate,
					marginCall.CallAmount,
					deliverAt); err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}
