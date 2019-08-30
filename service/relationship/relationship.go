package relationship

import (
	"fmt"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type RelationshipService interface {
	GetByID(accountID uuid.UUID, relID string) (*models.ACHRelationship, error)
	Create(accountID uuid.UUID, bInfo BankAcctInfo) (*models.ACHRelationship, error)
	Cancel(accountID uuid.UUID, relID string) error
	List(accountID uuid.UUID, statuses []enum.RelationshipStatus) ([]models.ACHRelationship, error)
	// Plaid interactions
	ExchangePlaidToken(publicToken string) (*plaid.Exchange, error)
	AuthPlaid(token string) (map[string]interface{}, error)
	GetPlaidItem(token string) (map[string]interface{}, error)
	GetPlaidInstitution(id string) (map[string]interface{}, error)
	WithTx(tx *gorm.DB) RelationshipService
	// Micro Deposit interactions
	Approve(accountID uuid.UUID, relID string, amountOne, amountTwo decimal.Decimal) (*models.ACHRelationship, error)
	Reissue(accountID uuid.UUID, relID string) (*models.ACHRelationship, error)
}

type relationshipService struct {
	RelationshipService
	tx                  *gorm.DB
	cancel              func(id string, reason string) (*apex.CancelRelationshipResponse, error)
	exchangeToken       func(publicToken string) (*plaid.Exchange, error)
	getAuth             func(token string) (map[string]interface{}, error)
	getItem             func(token string) (map[string]interface{}, error)
	getInstitution      func(id string) (map[string]interface{}, error)
	approveRelationship func(id string, amounts apex.MicroDepositAmounts) (*apex.ApproveRelationshipResponse, error)
	reissueMicroDeposit func(id string) error
}

func (s *relationshipService) WithTx(tx *gorm.DB) RelationshipService {
	s.tx = tx
	return s
}

func Service() RelationshipService {
	return &relationshipService{
		cancel:              apex.Client().CancelRelationship,
		exchangeToken:       plaid.Client().ExchangeToken,
		getAuth:             plaid.Client().GetAuth,
		getItem:             plaid.Client().GetItem,
		getInstitution:      plaid.Client().GetInstitution,
		approveRelationship: apex.Client().ApproveRelationship,
		reissueMicroDeposit: apex.Client().ReissueMicroDeposits,
	}
}

func (s *relationshipService) ExchangePlaidToken(publicToken string) (*plaid.Exchange, error) {
	return s.exchangeToken(publicToken)
}

func (s *relationshipService) AuthPlaid(token string) (map[string]interface{}, error) {
	return s.getAuth(token)
}

func (s *relationshipService) GetPlaidItem(token string) (map[string]interface{}, error) {
	return s.getItem(token)
}

func (s *relationshipService) GetPlaidInstitution(id string) (map[string]interface{}, error) {
	return s.getInstitution(id)
}

func (s *relationshipService) GetByID(accountID uuid.UUID, relID string) (*models.ACHRelationship, error) {
	rel := &models.ACHRelationship{}

	q := s.tx.Where("id = ? AND account_id = ?", relID, accountID).Find(&rel)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("relationship not found")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return rel, nil
}

type BankAcctInfo struct {
	Token       string
	Item        string
	Account     string
	Institution string
	BankAccount string
	Routing     string
	AccountType string
	Nickname    string
	Mask        string
	RelType     apex.ACHApprovalMethod
}

func (s *relationshipService) Create(accountID uuid.UUID, bInfo BankAcctInfo) (*models.ACHRelationship, error) {
	acct, err := getAccount(s.tx, accountID)
	if err != nil {
		return nil, err
	}

	if !utils.Dev() && !acct.Linkable() {
		return nil, gberrors.Unauthorized.WithMsg("account not authorized for linking with a bank")
	}

	expiresAt := clock.Now().Add(7 * calendar.Day)

	// Get the account's ACH Relationships. Check if any are queued or pending or approved status.
	// If there is -> reject
	// otherwise -> continue
	rels, err := s.List(accountID, []enum.RelationshipStatus{
		enum.RelationshipQueued,
		enum.RelationshipApproved,
		enum.RelationshipPending})
	if err != nil {
		return nil, err
	}

	if len(rels) > 0 {
		return nil, gberrors.Forbidden.WithMsg("active ach relationship already exists")
	}

	rel := &models.ACHRelationship{
		AccountID:        acct.ID,
		Status:           enum.RelationshipQueued,
		PlaidAccount:     &bInfo.Account,
		PlaidToken:       &bInfo.Token,
		PlaidItem:        &bInfo.Item,
		PlaidInstitution: &bInfo.Institution,
		Nickname:         &bInfo.Nickname,
		Mask:             &bInfo.Mask,
		ExpiresAt:        &expiresAt,
		ApprovalMethod:   bInfo.RelType,
	}

	bankInfo := models.BankInfo{
		Account:          bInfo.BankAccount,
		AccountOwnerName: *acct.Owners[0].Details.LegalName,
		RoutingNumber:    bInfo.Routing,
		AccountType:      bInfo.AccountType,
	}

	if err = rel.SetBankInfo(bankInfo); err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if err = s.tx.Create(rel).Error; err != nil {
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
			Institution string `json:"institution"`
		}{
			"bank_link_queued",
			*acct.ApexAccount,
			*acct.PrimaryOwner().Details.LegalName,
			acct.PrimaryOwner().Email,
			bInfo.Institution,
		})

		slack.Notify(msg)
	}

	return rel, nil
}

func (s *relationshipService) Cancel(accountID uuid.UUID, relID string) error {
	acct, err := getAccount(s.tx, accountID)
	if err != nil {
		return err
	}

	rel := &models.ACHRelationship{}

	if err = s.tx.Where(
		"id = ? AND account_id = ?",
		relID, acct.ID).
		First(rel).Error; err != nil {

		return gberrors.NotFound.WithError(fmt.Errorf("relationship not associated with account: %v", accountID))
	}

	// only cancel w/ apex if it has been approved by them
	if rel.Status == enum.RelationshipQueued {
		rel.Status = enum.RelationshipCanceled
	} else {
		resp, err := s.cancel(*rel.ApexID, "relationship canceled by account owner")
		if err != nil {
			log.Error("failed to cancel relationship with clearing broker", "relationship", relID, "error", err)
			return gberrors.InternalServerError.WithMsg("failed to cancel relationship with clearing broker")
		}

		rel.Status = enum.RelationshipStatus(*resp.Status)
	}

	transfers := []*models.Transfer{}

	if err = s.tx.Where(
		"relationship_id = ? AND status = ?",
		rel.ID, enum.TransferQueued).
		Set("gorm:query_option", db.ForUpdate).
		Find(&transfers).Error; err != nil {

		return gberrors.InternalServerError.WithError(err)
	}

	for i := range transfers {
		if err = s.tx.
			Model(transfers[i]).
			Update("status", enum.TransferCanceled).Error; err != nil {

			return gberrors.InternalServerError.WithError(err)
		}
	}

	// Check created_at. If it's less than 90 days from creation, update the Account to be flagged.
	// .Hours() gives hours since the 2 dates, so divide that by 24 to get days
	if (clock.Now().Sub(rel.CreatedAt).Hours() / 24) < 90 {
		d := date.DateOf(clock.Now().In(calendar.NY))
		acct.RiskyTransfers = true
		acct.MarkedRiskyTransfersAt = &d

		updates := map[string]interface{}{
			"risky_transfers":           true,
			"marked_risky_transfers_at": &d,
		}
		// Update the account to flag the account with risky_transfers == true
		if err = s.tx.Model(acct).Updates(updates).Error; err != nil {
			return gberrors.InternalServerError.WithError(err)
		}
	}

	// notify via slack
	if err = s.tx.Save(&rel).Error; err == nil {
		msg := slack.NewFundingActivity()

		msg.SetBody(struct {
			Type        string `json:"type"`
			ApexAccount string `json:"apex_account"`
			Name        string `json:"name"`
			Email       string `json:"email"`
			Institution string `json:"institution"`
		}{
			"bank_link_canceled",
			*acct.ApexAccount,
			*acct.PrimaryOwner().Details.LegalName,
			acct.PrimaryOwner().Email,
			*rel.PlaidInstitution,
		})

		slack.Notify(msg)
	}

	return err
}

// This won't list micro deposit accounts because those variables in the WHERE statements won't be populated
func (s *relationshipService) List(accountID uuid.UUID, statuses []enum.RelationshipStatus) ([]models.ACHRelationship, error) {
	rels := []models.ACHRelationship{}

	q := s.tx.
		Where("account_id = ?", accountID).
		Where("plaid_account IS NOT NULL").
		Where("plaid_token IS NOT NULL").
		Where("plaid_institution IS NOT NULL")

	if statuses != nil {
		q = q.Where("status IN (?)", statuses).Find(&rels)
	} else {
		q = q.Where("status != ?", apex.ACHCanceled).Find(&rels)
	}

	if q.Error != nil {
		return nil, q.Error
	}

	return rels, nil
}

func getAccount(tx *gorm.DB, accountID uuid.UUID) (*models.Account, error) {
	var acct models.Account

	q := tx.Where("id = ?", accountID).Find(&acct)
	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithError(fmt.Errorf("account not found"))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	if err := tx.Model(&acct).Related(&acct.Owners, "Owners").Error; err != nil {
		return nil, gberrors.NotFound.WithError(fmt.Errorf("primary owner not found"))
	}

	if err := tx.Model(&acct.Owners[0]).Related(&acct.Owners[0].Details).Error; err != nil {
		return nil, gberrors.NotFound.WithError(fmt.Errorf("primary owner details not found"))
	}
	return &acct, nil
}

func (s *relationshipService) Approve(accountID uuid.UUID, relID string, amountOne, amountTwo decimal.Decimal) (*models.ACHRelationship, error) {
	rel := &models.ACHRelationship{}

	if err := s.tx.Where(
		"id = ? AND account_id = ?",
		relID, accountID).
		First(rel).Error; err != nil {

		return nil, gberrors.NotFound.WithError(fmt.Errorf("relationship not associated with account: %v", accountID))
	}

	if rel.Status != enum.RelationshipPending {
		return nil, gberrors.Forbidden.WithMsg("bank relationship is not ready for approval")
	}

	if rel.FailedAttempts >= 3 {
		return rel, gberrors.Forbidden.WithMsg("failed too many times, please reissue micro deposits")
	}

	amounts := apex.MicroDepositAmounts{amountOne, amountTwo}
	zero := decimal.Zero
	one := decimal.New(1, 0)
	if amounts[0].LessThanOrEqual(zero) || amounts[0].GreaterThanOrEqual(one) || amounts[0].LessThanOrEqual(zero) || amounts[1].GreaterThanOrEqual(one) {
		return nil, gberrors.InvalidRequestParam.WithMsg("micro deposit amounts must be between 0 and 1")
	}

	if rel.ApexID == nil {
		return nil, gberrors.Forbidden.WithError(fmt.Errorf("no Apex ID associated with relationship %v", rel.ID))
	}

	resp, err := s.approveRelationship(*rel.ApexID, amounts)
	switch err {
	case nil:
		// Update the rel with the new status
		if err = s.tx.
			Model(rel).
			Update("status", enum.RelationshipStatus(*resp.Status)).Error; err != nil {

			return nil, gberrors.InternalServerError.WithError(err)
		}
		return rel, nil
	case apex.ErrInvalidAmounts:
		// update the database - need to handle it manually because an error is returned after
		tx := db.Begin()
		if err = tx.
			Model(rel).
			Update("failed_attempts", rel.FailedAttempts+1).Error; err != nil {
			tx.Rollback()
			return nil, gberrors.InternalServerError.WithError(err)
		}
		if err = tx.Commit().Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}
		return rel, gberrors.RequestBodyLoadFailure.WithMsg("micro deposit amounts do not match").WithError(err)
	default:
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("account approval failed for account: %v", accountID))
	}
}

func (s *relationshipService) Reissue(accountID uuid.UUID, relID string) (*models.ACHRelationship, error) {
	rel := &models.ACHRelationship{}

	// Check if the relationship is associated with the account
	if err := s.tx.Where(
		"id = ? AND account_id = ?",
		relID, accountID).
		First(rel).Error; err != nil {

		return nil, gberrors.NotFound.WithError(fmt.Errorf("relationship not associated with account: %v", accountID))
	}

	if rel.Status != enum.RelationshipPending {
		return nil, gberrors.Forbidden.WithMsg("cannot reissue micro deposits for this relationship")
	}

	// Check the ACH Relationships Failed Attempts >= 3
	if rel.FailedAttempts < 3 {
		return nil, gberrors.RequestBodyLoadFailure.WithMsg("can't reissue micro deposit until 3 failed attempts")
	}

	if rel.ApexID == nil {
		return nil, gberrors.Forbidden.WithError(fmt.Errorf("no Apex ID associated with relationship %v", rel.ID))
	}

	if err := s.reissueMicroDeposit(*rel.ApexID); err != nil {
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("micro deposit reissue failed for account: %v", accountID))
	}

	// Update FailedAttempts to 0
	if err := s.tx.
		Model(rel).
		Update("failed_attempts", 0).Error; err != nil {

		return nil, gberrors.InternalServerError.WithError(err)
	}
	return rel, nil
}
