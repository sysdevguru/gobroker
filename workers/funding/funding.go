package funding

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/service/relationship"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/workers/common"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type fundingWorker struct {
	createRelationship func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error)
	createTransfer     func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error)
	getBalance         func(rel *models.ACHRelationship) (*decimal.Decimal, error)
	done               chan struct{}
}

var worker *fundingWorker

func Work() {
	if worker == nil {
		worker = &fundingWorker{
			createRelationship: apex.Client().CreateRelationship,
			createTransfer:     apex.Client().Transfer,
			getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
				return plaid.Client().GetBalance(*rel.PlaidToken, *rel.PlaidAccount)
			},
			done: make(chan struct{}, 1),
		}
		worker.done <- struct{}{}
	}

	// make sure not to overlap if the work routine is taking long
	if common.WaitTimeout(worker.done, time.Second) {
		// timed out, so let's skip this round and wait until it finishes
		return
	}

	defer func() {
		worker.done <- struct{}{}
	}()

	// relationships
	worker.process(
		db.DB().Where("status = ?", enum.RelationshipQueued),
		&[]models.ACHRelationship{})

	// transfers
	worker.process(
		db.DB().
			Where("status IN (?)", []enum.TransferStatus{enum.TransferQueued, enum.TransferApprovalPending}).
			Preload("Relationship"),
		&[]models.Transfer{})
}

func (w *fundingWorker) process(q *gorm.DB, v interface{}) {
	if err := q.Find(v).Error; err != nil {
		log.Error("funding worker database error", "error", err)
		return
	}

	rows := reflect.ValueOf(v).Elem()

	for i := 0; i < rows.Len(); i++ {
		switch row := rows.Index(i).Interface().(type) {
		case models.ACHRelationship:
			w.handleRelationship(&row)
		case models.Transfer:
			w.handleTransfer(&row)
		}
	}
}

func (w *fundingWorker) handleRelationship(relationship *models.ACHRelationship) {
	tx := db.Begin()

	// handle panics to not leak DB transactions
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error("panicked while handling relationship", "relationship", relationship.ID)
		}
	}()

	if relationship.Expired() {
		log.Info(
			"canceling relationship due to expiration",
			"relationship", relationship.ID,
			"acct", relationship.AccountID)
		if err := tx.Model(relationship).Update("status", enum.RelationshipCanceled).Error; err != nil {
			tx.Rollback()
			log.Error(
				"failed to expire relationship",
				"relationship", relationship.ID,
				"acct", relationship.AccountID,
				"error", err)
			return
		}

		if err := tx.Commit().Error; err != nil {
			log.Error("funding worker database error", "error", err)
			return
		}

		return
	}

	srv := account.Service().WithTx(tx)

	acct, err := srv.GetByID(relationship.AccountIDAsUUID())
	if err != nil {
		tx.Rollback()
		log.Error(
			"failed to retrieve account for relationship",
			"relationship", relationship.ID,
			"account", acct.ID,
			"error", err)
		return
	}

	// ready to send to Apex
	if acct.Fundable() {
		payload, err := relationship.ForApex(acct)
		if err != nil {
			tx.Rollback()
			log.Error(
				"failed to generate payload ach relationship with apex",
				"relationship", relationship.ID,
				"account", acct.ID,
				"error", err)
			return
		}

		resp, err := w.createRelationship(*payload)
		if err != nil {
			tx.Rollback()
			log.Error(
				"failed to create ach relationship with clearing broker",
				"relationship", relationship.ID,
				"account", acct.ID,
				"error", err)
			return
		}

		updates := map[string]interface{}{
			"status":  enum.RelationshipStatus(*resp.Status),
			"apex_id": *resp.ID,
		}

		if err = tx.
			Model(relationship).
			Updates(updates).Error; err != nil {

			tx.Rollback()
			log.Error(
				"failed to update relationship status",
				"previous", relationship.Status,
				"new", *resp.Status,
				"relationship", relationship.ID,
				"account", acct.ID,
				"error", err)
			return
		}

		if err = tx.Commit().Error; err != nil {
			log.Error(
				"funding worker database error",
				"relationship", relationship.ID,
				"account", acct.ID,
				"error", err)
			return
		}

		// notify via slack
		{
			msg := slack.NewFundingActivity()

			msg.SetBody(struct {
				Type        string `json:"type"`
				ApexAccount string `json:"apex_account"`
				Name        string `json:"name"`
				Email       string `json:"email"`
				Institution string `json:"institution"`
			}{
				"bank_link_created",
				*acct.ApexAccount,
				*acct.PrimaryOwner().Details.LegalName,
				acct.PrimaryOwner().Email,
				*relationship.PlaidInstitution,
			})

			slack.Notify(msg)
		}

		return
	}

	// relationship isn't ready to be de-queued, rollback
	tx.Rollback()
}

func (w *fundingWorker) handleTransfer(transfer *models.Transfer) {
	tx := db.Begin()

	// handle panics to not leak DB transactions
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error("panicked while handling transfer", "transfer", transfer.ID, "account", transfer.AccountID)
		}
	}()

	if transfer.Expired() {
		log.Info(
			"canceling transfer due to expiration",
			"transfer", transfer.ID,
			"acct", transfer.AccountID)
		if err := tx.Model(transfer).Update("status", enum.TransferCanceled).Error; err != nil {
			tx.Rollback()
			log.Error(
				"failed to expire transfer",
				"transfer", transfer.ID,
				"acct", transfer.AccountID,
				"error", err)
			return
		}

		if err := tx.Commit().Error; err != nil {
			log.Error("funding worker database error", "error", err)
			return
		}

		return
	}

	if transfer.Status == enum.TransferApprovalPending {
		return
	}

	srv := account.Service().WithTx(tx)

	acct, err := srv.GetByID(transfer.AccountIDAsUUID())
	if err != nil {
		tx.Rollback()
		log.Error(
			"failed to retrieve account for transfer",
			"transfer", transfer.ID,
			"account", acct.ID,
			"error", err)
		return
	}

	// ready to send to apex
	if acct.Fundable() &&
		transfer.Relationship != nil &&
		transfer.Relationship.ApexID != nil &&
		transfer.Relationship.Status == enum.RelationshipApproved {

		// only check the balance in prod
		if utils.Prod() &&
			transfer.Direction == apex.Incoming &&
			transfer.Relationship.ApprovalMethod == apex.Plaid {

			balance, balanceErr := w.getBalance(transfer.Relationship)

			switch {
			case balanceErr != nil || balance == nil:
				// handle MFA/password change
				rel, err := relationship.Service().WithTx(tx).GetByID(transfer.AccountIDAsUUID(), *transfer.RelationshipID)
				if err != nil {
					tx.Rollback()
					return
				}
				err = handleBalanceError(tx, rel, transfer,
					relationship.Service().WithTx(tx).Cancel, balanceErr)
				if err != nil {
					tx.Rollback()
					return
				}

				return
			case balance.LessThan(transfer.Amount):
				if err := handleBalanceTooLow(tx, transfer); err != nil {
					tx.Rollback()
					return
				}

				if err = tx.Commit().Error; err != nil {
					log.Error(
						"funding worker database error",
						"relationship", *transfer.RelationshipID,
						"error", err)
				}

				return
			default:
				validated := true
				transfer.BalanceValidated = &validated
			}
		}

		ach := apex.ACHTransfer{
			ID:             transfer.ID,
			Amount:         transfer.Amount,
			RelationshipID: *transfer.Relationship.ApexID,
		}

		resp, err := w.createTransfer(transfer.Direction, ach)
		if err != nil {
			tx.Rollback()
			log.Error("failed to start transfer with clearing broker", "error", err)
			return
		}

		updates := map[string]interface{}{
			"apex_id": *resp.TransferID,
			"status":  enum.TransferStatus(*resp.State),
		}

		if err = tx.
			Model(transfer).
			Updates(updates).Error; err != nil {

			tx.Rollback()
			log.Error(
				"failed to update transfer apex_id",
				"apex_id", *resp.TransferID,
				"alpaca_id", transfer.ID,
				"error", err)
			return
		}

		if err = tx.Commit().Error; err != nil {
			log.Error("funding worker database error", "error", err)
			return
		}

		// notify via slack
		{
			msg := slack.NewFundingActivity()

			msg.SetBody(struct {
				Type        string `json:"type"`
				ApexAccount string `json:"apex_account"`
				Name        string `json:"name"`
				Email       string `json:"email"`
				Direction   string `json:"direction"`
				Amount      string `json:"amount"`
			}{
				"transfer_created",
				*acct.ApexAccount,
				*acct.PrimaryOwner().Details.LegalName,
				acct.PrimaryOwner().Email,
				string(transfer.Direction),
				transfer.Amount.String(),
			})

			slack.Notify(msg)
		}

		return
	}

	// transfer isn't ready to be de-queued, rollback
	tx.Rollback()
}

// Handles balance errors
func handleBalanceError(
	tx *gorm.DB,
	rel *models.ACHRelationship,
	transfer *models.Transfer,
	cancelRelationship func(accountID uuid.UUID, relID string) error,
	balanceErr error) error {

	if balanceErr != nil && strings.Contains(balanceErr.Error(), plaid.CodeItemLoginRequired) {
		err := tx.Model(transfer).Update("status", enum.TransferRejected).Error
		if err != nil {
			log.Error(
				"failed to reject transfer",
				"transfer", transfer.ID,
				"error", err)
			return err
		}

		err = cancelRelationship(
			transfer.AccountIDAsUUID(),
			*transfer.RelationshipID,
		)

		if err != nil {
			log.Error(
				"failed to cancel relationship",
				"relationship", *transfer.RelationshipID,
				"error", err)
			return err
		}

		// Get info for email
		acct, err := account.Service().WithTx(tx).GetByID(transfer.AccountIDAsUUID())
		if err != nil {
			log.Error(
				"failed to retrieve account associated with relationship",
				"relationship", *transfer.RelationshipID,
				"acct", transfer.AccountID,
				"error", err)
			return err
		}

		o := acct.PrimaryOwner()

		// Email to tell them to relink bank account due to MFA/Password change
		go mailer.SendPassMFAChange(
			*o.Details.GivenName,
			o.Email,
		)

		log.Info("transfer rejected due to MFA or password change",
			"acct", transfer.AccountID,
			"transfer", transfer.ID)

		return fmt.Errorf(plaid.CodeItemLoginRequired)

	} else {
		log.Error(
			"failed to retrieve plaid balance",
			"relationship", *transfer.RelationshipID,
			"error", balanceErr)
		return errors.Wrap(balanceErr, "plaid balance error")
	}
}

// Handles when bank balance is too low to transfer the designated amount
func handleBalanceTooLow(tx *gorm.DB, transfer *models.Transfer) error {

	if err := tx.Model(transfer).Update("status", enum.TransferRejected).Error; err != nil {
		log.Error(
			"failed to reject transfer",
			"transfer", transfer.ID,
			"error", err)
		return err
	}

	// Get info for email
	acct, err := account.Service().WithTx(tx).GetByID(transfer.AccountIDAsUUID())
	if err != nil {
		log.Error(
			"failed to retrieve account associated with relationship",
			"relationship", *transfer.RelationshipID,
			"acct", transfer.AccountID,
			"error", err)
		return err
	}

	o := acct.PrimaryOwner()

	// Email to tell them to check bank account
	go mailer.SendBalanceLow(
		*o.Details.GivenName,
		o.Email,
	)

	log.Info("transfer rejected due to balance too low",
		"acct", transfer.AccountID,
		"transfer", transfer.ID)

	return nil
}
