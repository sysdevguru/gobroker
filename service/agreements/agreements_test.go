package agreements

import (
	"fmt"
	"os"
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/external/polygon"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AgreementsTestSuite struct {
	dbtest.Suite
	acct *models.Account
}

func TestAgreementsTestSuite(t *testing.T) {
	suite.Run(t, new(AgreementsTestSuite))
}

func (s *AgreementsTestSuite) SetupSuite() {
	os.Setenv("BROKER_MODE", "DEV")

	s.SetupDB()

	amt, _ := decimal.NewFromString("1000000")
	apexAcct := "apca_test"
	legalName := "First Last"
	google := "Google Inc."
	position := "CEO"
	googleAddr := "1600 Amphitheatre Parkway, Mountain View, CA"
	employed := models.Employed
	function := "runs the place"
	city := "Somewhere"
	state := "SW"
	zip := "12345"

	s.acct = &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@example.com",
				Primary: true,
				Details: models.OwnerDetails{
					LegalName:        &legalName,
					Employer:         &google,
					EmployerAddress:  &googleAddr,
					EmploymentStatus: &employed,
					Position:         &position,
					Function:         &function,
					StreetAddress:    address.Address(pq.StringArray{"123 Somewhere Ln"}),
					City:             &city,
					State:            &state,
					PostalCode:       &zip,
				},
			},
		},
	}
	if err := db.DB().Create(s.acct).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *AgreementsTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AgreementsTestSuite) TestAgreements() {
	service := agreementsService{
		polygonAgreementFunc: func(name string, body interface{}) ([]byte, error) {
			return []byte("some pdf data"), nil
		},
		agreementStorageFunc: func(fileName string, data []byte) error {
			return nil
		},
	}

	srv := service.WithTx(db.DB())

	// successful
	{
		buf, err := srv.Get(s.acct.IDAsUUID(), polygon.NASDAQ)
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), buf)

		buf, err = srv.Get(s.acct.IDAsUUID(), polygon.NYSE)
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), buf)

		err = srv.Accept(s.acct.IDAsUUID(), polygon.NASDAQ)
		assert.Nil(s.T(), err)

		err = srv.Accept(s.acct.IDAsUUID(), polygon.NYSE)
		assert.Nil(s.T(), err)
	}

	// account not found
	{
		buf, err := srv.Get(uuid.Must(uuid.NewV4()), polygon.NYSE)
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), buf)

		err = srv.Accept(uuid.Must(uuid.NewV4()), polygon.NYSE)
		assert.NotNil(s.T(), err)
	}

	// No Address
	{
		// s.acct.Owners[0].Details.StreetAddress = address.Address(pq.StringArray{})
		buf, err := srv.Get(s.acct.IDAsUUID(), polygon.NASDAQ)
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), buf)

		s.T().Log(len(s.acct.Owners))

		od := models.OwnerDetails{}

		// Need to query the most recent Owner Details because it only updated the first one
		// that was created rather than doing the most recent for some reason
		err = service.tx.Where("replaced_by IS NULL").Find(&od).Error
		assert.Nil(s.T(), err)

		err = service.tx.Model(&od).Update("street_address", address.Address(pq.StringArray{})).Error
		assert.Nil(s.T(), err)

		buf, err = srv.Get(s.acct.IDAsUUID(), polygon.NYSE)
		assert.NotNil(s.T(), err)
		assert.Equal(s.T(), err, gberrors.Forbidden.WithMsg("format address failed because no address was given"))
		assert.Nil(s.T(), buf)
	}

	// polygon failure
	{
		service.polygonAgreementFunc = func(name string, body interface{}) ([]byte, error) {
			return nil, fmt.Errorf("polygon is dead")
		}

		srv = service.WithTx(db.DB())

		buf, err := srv.Get(s.acct.IDAsUUID(), polygon.NASDAQ)
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), buf)

		buf, err = srv.Get(s.acct.IDAsUUID(), polygon.NYSE)
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), buf)

		err = srv.Accept(s.acct.IDAsUUID(), polygon.NYSE)
		assert.NotNil(s.T(), err)
	}
}
