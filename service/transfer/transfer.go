package transfer

import (
	"fmt"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type TransferService interface {
	GetByID(accountID uuid.UUID, transferID string) (*models.Transfer, error)
	Create(accountID uuid.UUID, relID string, dir apex.TransferDirection, amt decimal.Decimal) (*models.Transfer, error)
	Cancel(accountID uuid.UUID, transferID string) error
	List(accountID uuid.UUID, dir *apex.TransferDirection, limit, offset *int) ([]models.Transfer, error)
	Update(transfer *models.Transfer) (*models.Transfer, error)
	WithTx(tx *gorm.DB) TransferService
}

type transferService struct {
	TransferService
	tx     *gorm.DB
	cancel func(id string, comment string) (*apex.CancelTransferResponse, error)
}

func Service() TransferService {
	return &transferService{
		cancel: apex.Client().CancelTransfer,
	}
}

func (s *transferService) WithTx(tx *gorm.DB) TransferService {
	s.tx = tx
	return s
}

func (s *transferService) GetByID(accountID uuid.UUID, transferID string) (*models.Transfer, error) {
	transfer := &models.Transfer{}

	q := s.tx.Where("id = ? AND account_id = ?", transferID, accountID).Find(&transfer)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("transfer not found")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return transfer, nil
}

func (s *transferService) Create(accountID uuid.UUID, relID string, dir apex.TransferDirection, amt decimal.Decimal) (*models.Transfer, error) {
	opt := db.ForUpdate

	acct, err := op.GetAccountByID(s.tx, accountID, &opt)
	if err != nil {
		return nil, err
	}

	if err = s.verifyTransfer(acct, amt, dir); err != nil {
		return nil, err
	}

	rel := &models.ACHRelationship{}

	q := s.tx.Where("id = ?", relID).First(rel)

	// make sure we have the relationship
	if q.RecordNotFound() {
		return nil, gberrors.InvalidRequestParam.WithMsg("relationship not found")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	expiresAt := clock.Now().Add(7 * calendar.Day)

	// Check if the account is flagged. If it is, update Status to be APPROVAL_PENDING
	var (
		status    enum.TransferStatus
		slackType string
	)

	if acct.RiskyTransfers && dir == apex.Outgoing {
		status = enum.TransferApprovalPending
		slackType = "transfer_approval_pending"
	} else {
		status = enum.TransferQueued
		slackType = "transfer_queued"
	}

	transfer := &models.Transfer{
		AccountID:      accountID.String(),
		RelationshipID: &relID,
		Type:           enum.ACH,
		Amount:         amt,
		Direction:      dir,
		Status:         status,
		ExpiresAt:      &expiresAt,
	}

	if err = s.tx.Create(transfer).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
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
			slackType,
			*acct.ApexAccount,
			*acct.PrimaryOwner().Details.LegalName,
			acct.PrimaryOwner().Email,
			string(dir),
			amt.String(),
		})

		slack.Notify(msg)
	}

	return transfer, err
}

func (s *transferService) Cancel(accountID uuid.UUID, transferID string) error {
	transfer := &models.Transfer{}

	opt := db.ForUpdate

	_, err := op.GetAccountByID(s.tx, accountID, &opt)
	if err != nil {
		return err
	}

	q := s.tx.Where("id = ? AND account_id = ?", transferID, accountID).Find(transfer)

	if q.RecordNotFound() {
		return gberrors.NotFound.WithError(fmt.Errorf("transfer not associated with account: %v", accountID))
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	if !transfer.Status.Cancelable() {
		return gberrors.InvalidRequestParam.WithMsg("transfer is not cancelable")
	}

	var status enum.TransferStatus

	if transfer.Status == enum.TransferQueued || transfer.Status == enum.TransferApprovalPending {
		status = enum.TransferCanceled
	} else {
		if transfer.ApexID == nil {
			return gberrors.InvalidRequestParam.WithMsg("too soon to cancel transfer")
		}

		resp, err := s.cancel(*transfer.ApexID, "cancellation requested by account owner")
		if err != nil {
			return gberrors.InternalServerError.
				WithMsg("failed to cancel transfer with clearing broker").
				WithError(err)
		}

		status = enum.TransferStatus(*resp.State)
	}

	return s.tx.Model(&transfer).Update("status", status).Error
}

func (s *transferService) List(accountID uuid.UUID, dir *apex.TransferDirection, limit, offset *int) ([]models.Transfer, error) {
	transfers := []models.Transfer{}

	q := s.tx.Where("account_id = ?", accountID)

	if dir != nil {
		q = q.Where("direction = ?", *dir)
	}

	if limit != nil {
		q = q.Limit(*limit)
	}

	if offset != nil {
		q = q.Offset(*offset)
	}

	q = q.Order("created_at DESC").Find(&transfers)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return transfers, nil
}

// dev purposes only
func (s *transferService) Update(transfer *models.Transfer) (*models.Transfer, error) {
	err := s.tx.Save(&transfer).Error
	return transfer, err
}

var transferLimit = decimal.New(50000, 0)

func (s *transferService) verifyTransfer(acct *models.Account, amt decimal.Decimal, dir apex.TransferDirection) error {
	if amt.LessThanOrEqual(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("transfer amount must be greater than $0")
	}

	// check transfer limit
	{
		todaysTransfers := []models.Transfer{}
		q := s.tx.
			Where("account_id = ? AND direction = ? AND created_at >= ?",
				acct.ID, dir, clock.Now().In(calendar.NY).Format("2006-01-02")).
			Find(&todaysTransfers)

		if q.Error != nil {
			return q.Error
		}

		totalAmt := amt
		for _, transfer := range todaysTransfers {
			totalAmt = totalAmt.Add(transfer.Amount)
		}

		if totalAmt.GreaterThan(transferLimit) {
			return gberrors.InvalidRequestParam.WithMsg("maximum total daily transfer allowed is $50,000")
		}
	}

	if dir == apex.Outgoing {
		tradeAccount, err := acct.ToTradeAccount()
		if err != nil {
			return err
		}

		if balances, err := op.GetAccountBalances(s.tx, tradeAccount); err != nil {
			return err
		} else if amt.GreaterThan(balances.CashWithdrawable) {
			return gberrors.Forbidden.WithMsg("transfer amount must be less than withdrawable cash")
		}
	}

	return nil
}
