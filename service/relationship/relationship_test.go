package relationship

import (
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RelationshipTestSuite struct {
	dbtest.Suite
	account *models.Account
}

func TestRelationshipTestSuite(t *testing.T) {
	suite.Run(t, new(RelationshipTestSuite))
}

func (s *RelationshipTestSuite) SetupSuite() {
	env.RegisterDefault("BROKER_SECRET", "fd0bxOTg7Q5qxISYKvdol0FBWnAaFgsP")
	env.RegisterDefault("BROKER_MODE", "DEV")

	s.SetupDB()
	amt, _ := decimal.NewFromString("10000")
	apexAcct := "apca_test"
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
			},
		},
	}
	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	legalName := "Test Trader"
	details := &models.OwnerDetails{
		OwnerID:   s.account.Owners[0].ID,
		LegalName: &legalName,
	}
	if err := db.DB().Create(details).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *RelationshipTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *RelationshipTestSuite) TestRelationship() {
	service := relationshipService{
		tx: db.DB(),
		cancel: func(id string, reason string) (*apex.CancelRelationshipResponse, error) {
			status := string(apex.ACHCanceled)
			return &apex.CancelRelationshipResponse{
				ID:     &id,
				Status: &status,
			}, nil
		},
		exchangeToken: func(publicToken string) (*plaid.Exchange, error) {
			return &plaid.Exchange{Token: "token", Item: "item"}, nil
		},
		getAuth: func(accessToken string) (map[string]interface{}, error) {
			return map[string]interface{}{"auth": "plaid"}, nil
		},
		getItem: func(accessToken string) (map[string]interface{}, error) {
			return map[string]interface{}{"item": "plaid"}, nil
		},
		getInstitution: func(id string) (map[string]interface{}, error) {
			return map[string]interface{}{"institution": "plaid"}, nil
		},
		approveRelationship: func(id string, amounts apex.MicroDepositAmounts) (*apex.ApproveRelationshipResponse, error) {
			statusOk := string(apex.ACHApproved)
			statusErr := string(apex.ACHPending)
			fail := decimal.New(11, -2)
			if amounts[0].Cmp(fail) == 0 && amounts[1].Cmp(fail) == 0 {
				return &apex.ApproveRelationshipResponse{
					ID:     &id,
					Status: &statusErr,
				}, apex.ErrInvalidAmounts
			}
			return &apex.ApproveRelationshipResponse{
				ID:     &id,
				Status: &statusOk,
			}, nil
		},
		reissueMicroDeposit: func(id string) error {
			return nil
		},
	}

	bInfo := BankAcctInfo{
		Token:       "token",
		Item:        "item",
		Account:     "account",
		Institution: "institution",
		BankAccount: "bank_account",
		Routing:     "routing",
		AccountType: "CHECKING",
		Nickname:    "my favorite checking account",
		RelType:     "plaid",
	}
	relationships, err := service.List(s.account.IDAsUUID(), nil)
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), relationships)

	relationship, err := service.Create(s.account.IDAsUUID(), bInfo)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	assert.Equal(s.T(), s.account.ID, relationship.AccountID)

	relationships, err = service.List(s.account.IDAsUUID(), nil)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), relationships, 1)

	exchange, err := service.ExchangePlaidToken("public_token")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), exchange)

	auth, _ := service.AuthPlaid("token")
	assert.NotNil(s.T(), auth)

	item, _ := service.GetPlaidItem("token")
	assert.NotNil(s.T(), item)

	inst, _ := service.GetPlaidInstitution("id")
	assert.NotNil(s.T(), inst)

	// Check ACHRelationships - Should result in an error saying that
	// an ach account already exists
	_, err = service.Create(s.account.IDAsUUID(), bInfo)
	assert.NotNil(s.T(), err)

	db.DB().Create(&models.Transfer{
		ID:             uuid.Must(uuid.NewV4()).String(),
		Type:           enum.ACH,
		AccountID:      s.account.ID,
		RelationshipID: &relationship.ID,
		Status:         enum.TransferQueued,
		Direction:      apex.Incoming,
	})

	// cancel queued internally
	err = service.Cancel(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)

	// cancel w/ apex
	relationship.Status = enum.RelationshipApproved
	relationship.ApexID = &relationship.ID
	require.Nil(s.T(), db.DB().Save(relationship).Error)

	err = service.Cancel(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)

	rel, err := service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)

	// Micro deposit tests

	// Create new relationship using micro deposit
	bInfo.RelType = "micro"
	relationship, err = service.Create(s.account.IDAsUUID(), bInfo)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	assert.Equal(s.T(), s.account.ID, relationship.AccountID)
	require.Nil(s.T(), db.DB().Model(relationship).Update("apex_id", &relationship.ID).Error)
	require.Nil(s.T(), db.DB().Model(relationship).Update("status", enum.RelationshipPending).Error)

	// Approve the amounts
	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(15, -2), decimal.New(45, -2))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), relationship.Status, enum.RelationshipApproved)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 0)

	// Fail 3 times (with a Reissue in the middle)
	require.Nil(s.T(), db.DB().Model(relationship).Update("status", enum.RelationshipPending).Error)
	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(11, -2), decimal.New(11, -2))
	assert.NotNil(s.T(), err)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 1)

	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(11, -2), decimal.New(11, -2))
	assert.NotNil(s.T(), err)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 2)

	// Try to Reissue before possible
	_, err = service.Reissue(s.account.IDAsUUID(), relationship.ID)
	assert.NotNil(s.T(), err)

	// Fail once more
	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(11, -2), decimal.New(11, -2))
	assert.NotNil(s.T(), err)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 3)

	// Test if it tells the user to reissue
	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(56, -2), decimal.New(44, -2))
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), err, gberrors.Forbidden.WithMsg("failed too many times, please reissue micro deposits"))

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 3)

	// Reissue the amounts
	relationship, err = service.Reissue(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), relationship)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 0)

	// Approve after the Reissue
	relationship, err = service.Approve(s.account.IDAsUUID(), relationship.ID, decimal.New(15, -2), decimal.New(45, -2))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), relationship.Status, enum.RelationshipApproved)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
	assert.Equal(s.T(), rel.FailedAttempts, 0)

	// Cancel with Apex
	relationship.Status = enum.RelationshipApproved
	relationship.ApexID = &relationship.ID
	require.Nil(s.T(), db.DB().Save(relationship).Error)

	err = service.Cancel(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)

	rel, err = service.GetByID(s.account.IDAsUUID(), relationship.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), rel)
}
