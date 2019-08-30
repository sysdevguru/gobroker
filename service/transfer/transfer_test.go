package transfer

import (
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TransferTestSuite struct {
	dbtest.Suite
	account      *models.Account
	relationship *models.ACHRelationship
}

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

func (s *TransferTestSuite) SetupSuite() {
	s.SetupDB()
	amt, _ := decimal.NewFromString("10000")
	apexAcct := "apca_test"
	legalName := "First Last"
	s.account = &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
				Details: models.OwnerDetails{
					LegalName: &legalName,
				},
			},
		},
	}
	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	plaidUUID := uuid.Must(uuid.NewV4()).String()
	s.relationship = &models.ACHRelationship{
		ID:               uuid.Must(uuid.NewV4()).String(),
		Status:           enum.RelationshipApproved,
		AccountID:        s.account.ID,
		ApprovalMethod:   apex.Plaid,
		PlaidAccount:     &plaidUUID,
		PlaidToken:       &plaidUUID,
		PlaidItem:        &plaidUUID,
		PlaidInstitution: &plaidUUID,
	}
	if err := db.DB().Create(s.relationship).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *TransferTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *TransferTestSuite) TestTransfer() {
	service := transferService{
		tx: db.DB(),
		cancel: func(id string, comment string) (*apex.CancelTransferResponse, error) {
			state := string(apex.TransferPending)
			return &apex.CancelTransferResponse{State: &state}, nil
		},
	}

	transfers, err := service.List(s.account.IDAsUUID(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), transfers)

	transfer, err := service.Create(
		s.account.IDAsUUID(),
		s.relationship.ID,
		apex.Incoming,
		decimal.NewFromFloat(1000),
	)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), transfer)
	assert.Equal(s.T(), s.account.ID, transfer.AccountID)

	transfers, err = service.List(s.account.IDAsUUID(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), transfers, 1)

	// simulate async ALE apexID
	apexID := uuid.Must(uuid.NewV4()).String()
	transfer.ApexID = &apexID
	service.tx.Save(transfer)

	// cancel queued
	assert.Nil(s.T(), service.Cancel(s.account.IDAsUUID(), transfer.ID))

	transfer.Status = enum.TransferPending
	require.Nil(s.T(), db.DB().Save(transfer).Error)
	// cancel w/ apex
	assert.Nil(s.T(), service.Cancel(s.account.IDAsUUID(), transfer.ID))

	xfer, err := service.GetByID(s.account.IDAsUUID(), transfer.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), xfer)

	// try to withdraw more than is available
	transfer, err = service.Create(
		s.account.IDAsUUID(),
		s.relationship.ID,
		apex.Outgoing,
		decimal.NewFromFloat(10001),
	)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), transfer)

	// withdraw normally
	transfer, err = service.Create(
		s.account.IDAsUUID(),
		s.relationship.ID,
		apex.Outgoing,
		decimal.NewFromFloat(5000),
	)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), transfer)

	transfers, err = service.List(s.account.IDAsUUID(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), transfers, 2)

	// try deposit more than $50k total
	transfer, err = service.Create(
		s.account.IDAsUUID(),
		s.relationship.ID,
		apex.Incoming,
		decimal.New(49001, 0),
	)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "maximum total daily transfer allowed is $50,000")
	assert.Nil(s.T(), transfer)

	// $0 transfer
	transfer, err = service.Create(
		s.account.IDAsUUID(),
		s.relationship.ID,
		apex.Incoming,
		decimal.New(0, 0),
	)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "transfer amount must be greater than $0")
	assert.Nil(s.T(), transfer)
}
