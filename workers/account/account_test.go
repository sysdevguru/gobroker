package account

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/apex/forms"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/rmq/pubsub"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AccountWorkerTestSuite struct {
	dbtest.Suite
	asset *models.Asset
}

func TestAccountWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountWorkerTestSuite))
}

func (s *AccountWorkerTestSuite) SetupSuite() {
	env.RegisterDefault("BROKER_SECRET", "fd0bxOTg7Q5qxISYKvdol0FBWnAaFgsP")
	env.RegisterDefault("BROKER_MODE", "DEV")
	s.SetupDB()
}

func (s *AccountWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AccountWorkerTestSuite) TestAccountWorker() {

	// initialize worker
	worker = &accountWorker{
		stream: make(chan pubsub.Message, 10),
		apexPostAcct: func(sub forms.FormSubmission) (arr *apex.AccountRequestResponse, body []byte, err error) {

			apexAcct := "apex_account"
			status := string(enum.Pending)
			id := uuid.Must(uuid.NewV4()).String()

			return &apex.AccountRequestResponse{
				Account: &apexAcct,
				Status:  &status,
				ID:      &id,
			}, nil, nil
		},
		apexGetAcct: func(requestId string) (*apex.AccountRequestResponse, error) {

			apexAcct := "apex_account"
			status := string(enum.Pending)
			id := uuid.Must(uuid.NewV4()).String()

			return &apex.AccountRequestResponse{
				Account:   &apexAcct,
				Status:    &status,
				ID:        &id,
				SketchIDs: []string{uuid.Must(uuid.NewV4()).String()},
			}, nil
		},
		apexGetSketch: func(id string) (*apex.GetSketchInvestigationResponse, error) {
			status := string(models.SketchPending)
			return &apex.GetSketchInvestigationResponse{Status: &status}, nil
		},
	}

	// onboarding

	// create account
	acctSrv := account.Service().WithTx(db.DB())
	acct, err := acctSrv.Create(
		"test@example.com",
		uuid.Must(uuid.NewV4()),
	)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), acct)

	// patch owner details
	patches := map[string]interface{}{}
	err = json.Unmarshal([]byte(details), &patches)
	require.Nil(s.T(), err)

	var apexAddr address.Address
	for _, val := range patches["street_address"].([]interface{}) {
		apexAddr = append(apexAddr, val.(string))
	}
	patches["street_address"] = apexAddr

	detSrv := ownerdetails.Service().WithTx(db.DB())
	detSrv.Patch(acct.IDAsUUID(), patches)
	details, err := detSrv.GetPrimaryByAccountID(acct.IDAsUUID())
	require.Nil(s.T(), err)

	// handle onboarding
	worker.handleOnboarding(db.DB(), acct, details)

	// confirm account was onboarded
	acct, _ = acctSrv.GetByID(acct.IDAsUUID())
	assert.Equal(s.T(), enum.Submitted, acct.Status)

	// account updated

	// make the account active
	acct, err = acctSrv.Patch(acct.IDAsUUID(), map[string]interface{}{"status": enum.Active})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), acct)

	// update an attribute
	details, err = detSrv.Patch(acct.IDAsUUID(), map[string]interface{}{"phone_number": "555-555-5555"})
	require.Nil(s.T(), err)

	// ensure status is updated
	acct, _ = acctSrv.GetByID(acct.IDAsUUID())
	assert.Equal(s.T(), enum.AccountUpdated, acct.Status)
	worker.handleAccountUpdated(db.DB(), acct, details)

	// ensure resubmission
	acct, _ = acctSrv.GetByID(acct.IDAsUUID())
	assert.Equal(s.T(), enum.Resubmitted, acct.Status)

	// action required
	// worker.handleActionRequired(db.DB(), acct)

	// complete

	// dev mode - straight to active
	worker.handleComplete(db.DB(), acct)
	assert.Equal(s.T(), enum.Active, acct.Status)

	// stg/prod - approval_pending
	acct.Status = enum.Submitted
	os.Setenv("BROKER_MODE", "STG")
	worker.handleComplete(db.DB(), acct)
	assert.Equal(s.T(), enum.ApprovalPending, acct.Status)
}

var details = `
{
  "prefix": "",
  "given_name": "Name0",
  "additional_name": "",
  "family_name": "Family0",
  "suffix": "",
  "legal_name": "Name0 Family0",
  "date_of_birth": "1978-01-01",
  "ssn": "666-00-0001",
  "phone_number": "666-666-6666",
  "street_address": [
    "East 5th Street"
  ],
  "unit": null,
  "city": "Manhattan",
  "state": "NY",
  "postal_code": "10009",
  "country_of_citizenship": "USA",
  "permanent_resident": true,
  "visa_type": "",
  "employment_status": "UNEMPLOYED",
  "marital_status": "MARRIED",
  "number_of_dependents": 0,
  "is_control_person": false,
  "is_affiliated_exchange_or_finra": false,
  "is_politically_exposed": false,
  "margin_agreement_signed": true,
  "account_agreement_signed": true
}
`
