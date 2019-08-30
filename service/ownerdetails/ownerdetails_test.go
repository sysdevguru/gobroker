package ownerdetails

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OwnerDetailsTestSuite struct {
	dbtest.Suite
}

func TestOwnerDetailsTestSuite(t *testing.T) {
	suite.Run(t, new(OwnerDetailsTestSuite))
}

func (s *OwnerDetailsTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *OwnerDetailsTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *OwnerDetailsTestSuite) TestOwnerDetails() {
	// Account creation

	tx := db.Begin()
	srv := account.Service().WithTx(tx)

	acct, _ := srv.Create(
		"test+x@example.com",
		uuid.Must(uuid.NewV4()),
	)
	assert.NotNil(s.T(), acct)
	require.Nil(s.T(), tx.Commit().Error)

	// detail updates (onboarding)
	{
		tx := db.Begin()

		odsrv := ownerDetailsService{tx: tx}

		od, err := odsrv.GetPrimaryByAccountID(acct.IDAsUUID())
		require.Nil(s.T(), err)
		assert.NotNil(s.T(), od)

		odsrv = ownerDetailsService{tx: tx}

		od, err = odsrv.Patch(
			acct.IDAsUUID(),
			map[string]interface{}{"date_of_birth": "1991-01-01"},
		)
		require.Nil(s.T(), err)
		assert.NotNil(s.T(), od)

		od, err = odsrv.Patch(
			acct.IDAsUUID(),
			map[string]interface{}{
				"street_address": address.Address([]string{"some", "random", "address"}),
			},
		)
		require.Nil(s.T(), err)
		assert.NotNil(s.T(), od)

		require.Nil(s.T(), tx.Commit().Error)

		// Confirm it is updated
		odsrv = ownerDetailsService{tx: db.DB()}
		od, err = odsrv.GetPrimaryByAccountID(acct.IDAsUUID())
		require.Nil(s.T(), err)
		assert.Equal(s.T(), *od.DateOfBirthString(), "1991-01-01")

		tx = db.Begin()

		// Details update country_of_birth with nil
		odsrv = ownerDetailsService{tx: tx}

		od, _ = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"country_of_birth": "JPN"})
		assert.NotNil(s.T(), od)
		assert.Equal(s.T(), "JPN", *od.CountryOfBirth)

		od, _ = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"country_of_birth": nil})
		assert.NotNil(s.T(), od)
		assert.Nil(s.T(), od.CountryOfBirth)
		require.Nil(s.T(), tx.Commit().Error)
	}

	// detail updates (post approval)
	{
		tx := db.Begin()

		apexAcct := "APEX_ACCT"
		acct.ApexAccount = &apexAcct
		require.Nil(s.T(), db.DB().Save(&acct).Error)

		odsrv := ownerDetailsService{tx: db.DB()}

		od, err := odsrv.GetPrimaryByAccountID(acct.IDAsUUID())
		require.Nil(s.T(), err)
		assert.NotNil(s.T(), od)

		odsrv = ownerDetailsService{tx: tx}

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"state": "NY"})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Equal(s.T(), "NY", *od.State)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"city": "Bronx"})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Equal(s.T(), "Bronx", *od.City)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"unit": "1"})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Equal(s.T(), "1", *od.Unit)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"postal_code": "94402"})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Equal(s.T(), "94402", *od.PostalCode)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"phone_number": "111-111-1111"})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Equal(s.T(), "111-111-1111", *od.PhoneNumber)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{
			"street_address": address.Address([]string{"123 Somewhere Ln."}),
		})
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), od)
		assert.Len(s.T(), od.StreetAddress, 1)

		t := time.Now()

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"nasdaq_agreement_signed_at": t})
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), t, *od.NasdaqAgreementSignedAt)

		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"nyse_agreement_signed_at": t})
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), t, *od.NyseAgreementSignedAt)

		// disallowed field
		od, err = odsrv.Patch(acct.IDAsUUID(), map[string]interface{}{"owner_id": "1"})
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), od)

		require.Nil(s.T(), tx.Commit().Error)
	}
}
