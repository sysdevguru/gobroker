package funding

import (
	"encoding/json"
	"fmt"
	"github.com/alpacahq/gobroker/external/plaid"
	"testing"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var plaidUUID = uuid.Must(uuid.NewV4()).String()

type FundingWorkerTestSuite struct {
	dbtest.Suite
	account  *models.Account
	bankInfo []byte
}

func TestFundingWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(FundingWorkerTestSuite))
}

func (s *FundingWorkerTestSuite) SetupSuite() {
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
	givenName := "Test"
	familyName := "Trader"
	details := &models.OwnerDetails{
		OwnerID:    s.account.Owners[0].ID,
		GivenName:  &givenName,
		FamilyName: &familyName,
		LegalName:  &legalName,
	}
	if err := db.DB().Create(details).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	buf, err := json.Marshal(models.BankInfo{
		Account:          "12343456789",
		AccountOwnerName: "Test Trader",
		RoutingNumber:    "123456789",
		AccountType:      "CHECKING",
	})

	require.Nil(s.T(), err)

	s.bankInfo, err = encryption.EncryptWithKey(
		buf, []byte(env.GetVar("BROKER_SECRET")))

	require.Nil(s.T(), err)
}

func (s *FundingWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *FundingWorkerTestSuite) TestFundingWorker() {
	// relationships
	{
		// successful queue process
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			require.Nil(s.T(), db.DB().Create(&models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// queued, but account not ready
		{
			db.DB().Model(s.account).Update("apex_approval_status", enum.Suspended)

			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			require.Nil(s.T(), db.DB().Create(&models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// apex fails to create relationship
		{
			db.DB().Model(s.account).Update("apex_approval_status", enum.Complete)

			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					return nil, fmt.Errorf("apex can't write code")
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			require.Nil(s.T(), db.DB().Create(&models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// expired
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			exp := clock.Now().Add(-time.Minute)
			require.Nil(s.T(), db.DB().Create(&models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
				ExpiresAt:        &exp,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}
	}

	// transfers
	{
		// successful queue process
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipApproved,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}
			require.Nil(s.T(), db.DB().Create(rel).Error)

			require.Nil(s.T(), db.DB().Create(&models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// queued, but account not ready
		{
			db.DB().Model(s.account).Update("status", enum.Rejected)

			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			apexID := uuid.Must(uuid.NewV4()).String()
			worker.done <- struct{}{}
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				ApexID:           &apexID,
				Status:           enum.RelationshipApproved,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}
			require.Nil(s.T(), db.DB().Create(rel).Error)

			require.Nil(s.T(), db.DB().Create(&models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// queued, account ready, but relationship not ready
		{
			db.DB().Model(s.account).Update("status", enum.Active)

			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}
			require.Nil(s.T(), db.DB().Create(rel).Error)

			require.Nil(s.T(), db.DB().Create(&models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// insufficient balance
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(1, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipQueued,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}

			tx := db.DB()
			require.Nil(s.T(), tx.Create(rel).Error)

			transfer := &models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}

			require.Nil(s.T(), tx.Create(transfer).Error)

			assert.NotPanics(s.T(), Work)

			err := handleBalanceTooLow(tx, transfer)
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), transfer.Status, enum.TransferRejected)
		}

		// apex fails to create transfer
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					return nil, fmt.Errorf("apex can't code")
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			apexID := uuid.Must(uuid.NewV4()).String()
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				ApexID:           &apexID,
				Status:           enum.RelationshipApproved,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}
			require.Nil(s.T(), db.DB().Create(rel).Error)

			require.Nil(s.T(), db.DB().Create(&models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// expired
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			apexID := uuid.Must(uuid.NewV4()).String()
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				ApexID:           &apexID,
				Status:           enum.RelationshipApproved,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}
			require.Nil(s.T(), db.DB().Create(rel).Error)

			exp := clock.Now().Add(-time.Minute)
			require.Nil(s.T(), db.DB().Create(&models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
				ExpiresAt:      &exp,
			}).Error)

			assert.NotPanics(s.T(), Work)
		}

		// MFA/Password Change
		{
			worker = &fundingWorker{
				createRelationship: func(rel apex.ACHRelationship) (*apex.CreateRelationshipResponse, error) {
					pending := string(apex.ACHPending)
					id := uuid.Must(uuid.NewV4()).String()
					return &apex.CreateRelationshipResponse{
						Status: &pending,
						ID:     &id,
					}, nil
				},
				createTransfer: func(dir apex.TransferDirection, transfer apex.ACHTransfer) (*apex.TransferResponse, error) {
					id := "some_transfer_id"
					state := string(enum.TransferPending)
					return &apex.TransferResponse{
						TransferID: &id,
						State:      &state,
					}, nil
				},
				getBalance: func(rel *models.ACHRelationship) (*decimal.Decimal, error) {
					bal := decimal.New(100000, 0)
					return &bal, nil
				},
				done: make(chan struct{}, 1),
			}
			worker.done <- struct{}{}
			rel := &models.ACHRelationship{
				ID:               uuid.Must(uuid.NewV4()).String(),
				Status:           enum.RelationshipApproved,
				ApexID:           &s.account.ID,
				AccountID:        s.account.ID,
				ApprovalMethod:   apex.Plaid,
				PlaidAccount:     &plaidUUID,
				PlaidToken:       &plaidUUID,
				PlaidItem:        &plaidUUID,
				PlaidInstitution: &plaidUUID,
				HashBankInfo:     s.bankInfo,
			}

			tx := db.DB()
			require.Nil(s.T(), tx.Create(rel).Error)

			transfer := &models.Transfer{
				ID:             uuid.Must(uuid.NewV4()).String(),
				Type:           enum.ACH,
				AccountID:      s.account.ID,
				RelationshipID: &rel.ID,
				Amount:         decimal.New(1000, 0),
				Status:         enum.TransferQueued,
				Direction:      apex.Incoming,
			}

			require.Nil(s.T(), tx.Create(transfer).Error)

			assert.NotPanics(s.T(), Work)

			cancelRelationship := func(accountID uuid.UUID, relID string) error {
				err := tx.Model(rel).Update("status", enum.RelationshipCanceled).Error
				return err
			}

			// getBalance returns an error
			errStr := fmt.Errorf("other error")
			err := handleBalanceError(
				tx, rel, transfer,
				cancelRelationship,
				errStr,
			)
			assert.NotNil(s.T(), err)
			assert.Equal(s.T(), rel.Status, enum.RelationshipApproved)
			assert.Equal(s.T(), transfer.Status, enum.TransferQueued)

			errStr = fmt.Errorf(plaid.CodeItemLoginRequired)
			err = handleBalanceError(
				tx, rel, transfer,
				cancelRelationship,
				errStr,
			)
			assert.Equal(s.T(), err, errStr)
			assert.Equal(s.T(), rel.Status, enum.RelationshipCanceled)
			assert.Equal(s.T(), transfer.Status, enum.TransferRejected)
		}
	}
}
