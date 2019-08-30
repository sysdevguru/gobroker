package ale

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/relationship"
	"github.com/alpacahq/gobroker/service/transfer"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ALEWorkerTestSuite struct {
	dbtest.Suite
	asset    *models.Asset
	apexAcct string
}

func TestALEWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(ALEWorkerTestSuite))
}

func (s *ALEWorkerTestSuite) SetupSuite() {
	os.Setenv("BROKER_MODE", "DEV")
	s.apexAcct = "apca_test"
	s.SetupDB()
}

func (s *ALEWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *ALEWorkerTestSuite) TestALEWorker() {
	acctSrv := account.Service().WithTx(db.DB())

	acct, err := acctSrv.Create(
		"test@example.com",
		uuid.Must(uuid.NewV4()),
	)
	require.Nil(s.T(), err)

	require.Nil(s.T(), db.DB().Model(acct).Update("apex_account", s.apexAcct).Error)

	// acccount requests
	{
		acct, err = acctSrv.Patch(
			acct.IDAsUUID(),
			map[string]interface{}{"apex_request_id": uuid.Must(uuid.NewV4()).String()},
		)
		require.Nil(s.T(), err)

		worker = &aleWorker{
			apexGetAcctReq: func(requestId string) (*apex.AccountRequestResponse, error) {
				return &apex.AccountRequestResponse{
					SketchIDs: []string{uuid.Must(uuid.NewV4()).String()},
				}, nil
			},
			apexGetSketch: func(id string) (*apex.GetSketchInvestigationResponse, error) {
				status := string(models.SketchAccepted)

				req := struct {
					Identity *struct {
						Name *struct {
							Prefix          interface{}   `json:"prefix"`
							GivenName       *string       `json:"givenName"`
							AdditionalNames []interface{} `json:"additionalNames"`
							FamilyName      *string       `json:"familyName"`
							Suffix          interface{}   `json:"suffix"`
						} `json:"name"`
						HomeAddress *struct {
							StreetAddress []string `json:"streetAddress"`
							City          *string  `json:"city"`
							State         *string  `json:"state"`
							PostalCode    *string  `json:"postalCode"`
							Country       *string  `json:"country"`
						} `json:"homeAddress"`
						MailingAddress *struct {
							StreetAddress []string `json:"streetAddress"`
							City          *string  `json:"city"`
							State         *string  `json:"state"`
							PostalCode    *string  `json:"postalCode"`
							Country       *string  `json:"country"`
						} `json:"mailingAddress"`
						PhoneNumber          interface{} `json:"phoneNumber"`
						SocialSecurityNumber *string     `json:"socialSecurityNumber"`
						CitizenshipCountry   *string     `json:"citizenshipCountry"`
						DateOfBirth          *string     `json:"dateOfBirth"`
					} `json:"identity"`
					IncludeIdentityVerification *bool   `json:"includeIdentityVerification"`
					CorrespondentCode           *string `json:"correspondentCode"`
					Branch                      *string `json:"branch"`
					Account                     *string `json:"account"`
					Source                      *string `json:"source"`
					SourceID                    *string `json:"sourceId"`
				}{
					Account: &s.apexAcct,
				}

				return &apex.GetSketchInvestigationResponse{
					Status:  &status,
					Request: &req,
				}, nil
			},
		}

		buf, _ := json.Marshal(map[string]interface{}{
			"requestId": acct.ApexRequestID,
			"status":    string(enum.BackOffice),
		})

		update := apex.ALEMessage{
			Payload: string(buf),
		}

		err = worker.accountUpdateHandler(db.DB(), update)
		assert.Nil(s.T(), err)

		acct, err = acctSrv.GetByID(acct.IDAsUUID())
		require.Nil(s.T(), err)
		assert.True(s.T(), enum.BackOffice == acct.ApexApprovalStatus)

		// When an account gets rejected
		buf, _ = json.Marshal(map[string]interface{}{
			"requestId": acct.ApexRequestID,
			"status":    string(enum.ApexRejected),
		})

		update = apex.ALEMessage{
			Payload: string(buf),
		}

		err = worker.accountUpdateHandler(db.DB(), update)
		assert.Nil(s.T(), err)

		acct, err = acctSrv.GetByID(acct.IDAsUUID())
		require.Nil(s.T(), err)
		assert.True(s.T(), enum.ApexRejected == acct.ApexApprovalStatus)
		assert.Equal(s.T(), acct.Status, enum.Rejected)
		// assert.Nil(s.T(), acct.ApexAccount)
	}

	amt, _ := decimal.NewFromString("10000")
	legalName := "First Last"
	s.apexAcct = "apca_test0"
	acct = &models.Account{
		ApexAccount:        &s.apexAcct,
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
	require.Nil(s.T(), db.DB().Create(acct).Error)

	bInfo := relationship.BankAcctInfo{
		Token:       "token",
		Item:        "item",
		Account:     "account",
		Institution: "institution",
		BankAccount: "bank_account",
		Routing:     "routing",
		AccountType: "CHECKING",
		Nickname:    "my favorite checking account",
	}
	id := uuid.Must(uuid.NewV4()).String()
	rel := &models.ACHRelationship{
		ID:               id,
		ApexID:           &id,
		AccountID:        acct.ID,
		Status:           enum.RelationshipApproved,
		ApprovalMethod:   apex.Plaid,
		PlaidAccount:     &bInfo.Account,
		PlaidToken:       &bInfo.Token,
		PlaidItem:        &bInfo.Item,
		PlaidInstitution: &bInfo.Institution,
	}
	require.Nil(s.T(), db.DB().Create(rel).Error)

	xferSrv := transfer.Service().WithTx(db.DB())

	xfer, err := xferSrv.Create(
		acct.IDAsUUID(),
		rel.ID,
		apex.Outgoing,
		decimal.NewFromFloat(100))
	require.Nil(s.T(), err)

	// transfer status
	{
		worker.apexTransferStatus = func(id string) (*apex.TransferStatusResponse, error) {
			direction := string(xfer.Direction)
			state := string(apex.TransferFundsPosted)
			eta := "2018-04-14"
			return &apex.TransferStatusResponse{
				ExternalTransferID:          &xfer.ID,
				AchRelationshipID:           &rel.ID,
				Amount:                      &amt,
				Direction:                   &direction,
				State:                       &state,
				EstimatedFundsAvailableDate: &eta,
			}, nil
		}

		buf, _ := json.Marshal(map[string]interface{}{
			"externalTransferId": xfer.ID,
			"account":            *acct.ApexAccount,
			"transferId":         xfer.ID,
		})

		update := apex.ALEMessage{
			Payload: string(buf),
		}

		err = worker.transferUpdateHandler(db.DB(), update)
		assert.Nil(s.T(), err)
	}

	// micro status
	{
		buf, _ := json.Marshal(map[string]interface{}{
			"account":           *acct.ApexAccount,
			"achRelationshipId": *rel.ApexID,
			"status":            enum.TransferFundsPosted,
		})

		update := apex.ALEMessage{
			Payload: string(buf),
		}

		assert.Nil(s.T(), worker.microUpdateHandler(db.DB(), update))

	}

	// relationship status
	{
		worker.apexGetRel = func(id string) (*apex.GetRelationshipResponse, error) {
			status := string(enum.RelationshipApproved)
			method := string(apex.Plaid)
			return &apex.GetRelationshipResponse{
				Account:        acct.ApexAccount,
				Status:         &status,
				ApprovalMethod: &method,
			}, nil
		}

		buf, _ := json.Marshal(map[string]interface{}{
			"relationshipId": rel.ID,
			"status":         string(enum.RelationshipApproved),
		})
		update := apex.ALEMessage{
			Payload: string(buf),
		}

		err = worker.relationshipUpdateHandler(db.DB(), update)
		assert.Nil(s.T(), err)
	}

	// sketch status
	{
		require.Nil(s.T(), db.DB().
			Model(acct).
			Update("apex_request_id", uuid.Must(uuid.NewV4()).String()).Error)

		buf, err := json.Marshal(map[string]interface{}{
			"state":     string(models.SketchAccepted),
			"requestId": acct.ApexRequestID,
		})
		require.Nil(s.T(), err)
		update := apex.ALEMessage{
			Payload: string(buf),
		}

		err = worker.sketchHandler(db.DB(), update)
		assert.Nil(s.T(), err)
	}

	snap := &models.Snap{
		ID:                uuid.Must(uuid.NewV4()).String(),
		AccountID:         acct.ID,
		MimeType:          "image/png",
		Name:              models.DriverLicense.String(),
		DocumentRequestID: uuid.Must(uuid.NewV4()).String(),
	}
	require.Nil(s.T(), db.DB().Create(snap).Error)

	// snap status
	{
		buf, _ := json.Marshal(map[string]interface{}{
			"id": snap.ID,
		})
		update := apex.ALEMessage{
			Payload: string(buf),
		}
		err = worker.snapHandler(db.DB(), update)
		assert.Nil(s.T(), err)
	}

	hermesFailure := &models.HermesFailure{
		ID:                uuid.Must(uuid.NewV4()).String(),
		Status:            apex.HermesResend,
		Email:             "apex@trader.com",
		CorrespondentCode: "APCC",
	}
	require.Nil(s.T(), db.DB().Create(hermesFailure).Error)

	// hermes status
	{
		buf, _ := json.Marshal(map[string]interface{}{
			"notificationId":    hermesFailure.ID,
			"correspondentCode": "APCC",
			"email":             "apex@trader.com",
			"status":            hermesFailure.Status,
		})
		update := apex.ALEMessage{
			Payload: string(buf),
		}
		err = worker.hermesHandler(db.DB(), update)
		assert.Nil(s.T(), err)
	}
}
