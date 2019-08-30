package trustedcontact

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TrustedContactTestSuite struct {
	dbtest.Suite
}

func TestTrustedContactTestSuite(t *testing.T) {
	suite.Run(t, new(TrustedContactTestSuite))
}

func (s *TrustedContactTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *TrustedContactTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *TrustedContactTestSuite) TestTrustedContact() {
	srv := trustedContactService{tx: db.DB()}

	acctID := uuid.Must(uuid.NewV4())
	tc, err := srv.GetByID(acctID)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), tc)

	email := "trader@test.com"
	phone := "1112223333"
	city := "Somewhere"
	state := "SW"
	postal := "12345"
	country := "USA"

	tc = &models.TrustedContact{
		AccountID:     acctID.String(),
		EmailAddress:  &email,
		PhoneNumber:   &phone,
		StreetAddress: address.Address{"123 Somewhere Ln"},
		City:          &city,
		State:         &state,
		PostalCode:    &postal,
		Country:       &country,
		GivenName:     "Test",
		FamilyName:    "Trader",
	}

	tc, err = srv.Create(tc)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tc)
	assert.Equal(s.T(), acctID.String(), tc.AccountID)

	// creating a second time should be fine
	tc = &models.TrustedContact{
		AccountID:     acctID.String(),
		EmailAddress:  &email,
		PhoneNumber:   &phone,
		StreetAddress: address.Address{"123 Somewhere Ln"},
		City:          &city,
		State:         &state,
		PostalCode:    &postal,
		Country:       &country,
		GivenName:     "Changed",
		FamilyName:    "Trader",
	}
	tc, err = srv.Upsert(tc)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tc)

	tc, err = srv.GetByID(acctID)
	assert.Equal(s.T(), "Changed", tc.GivenName)

	email = "test@trader.com"
	phone = "3332221111"
	city = "Nowhere"
	state = "NW"
	postal = "99999"
	country = "NWR"
	street := []string{"123 Nowhere Blvd", "Apt 3", "PO Box 12345"}

	patches := map[string]interface{}{
		"email_address":  email,
		"phone_number":   phone,
		"street_address": street,
		"city":           city,
		"state":          state,
		"postal_code":    postal,
		"country":        "country",
		"given_name":     "Trader",
		"family_name":    "Test",
	}

	tc, err = srv.Patch(acctID, patches)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tc)
	assert.Equal(s.T(), email, *tc.EmailAddress)
	assert.Equal(s.T(), phone, *tc.PhoneNumber)
	assert.Equal(s.T(), "Test", tc.FamilyName)

	patches = map[string]interface{}{
		"email_address":  email,
		"phone_number":   phone,
		"street_address": "123 Nowhere Blvd Apt 3 PO Box 12345",
		"city":           city,
		"state":          state,
		"postal_code":    postal,
		"country":        "country",
		"given_name":     "Trader",
		"family_name":    "Test",
	}

	tc, err = srv.Patch(acctID, patches)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tc)
	assert.Equal(s.T(), email, *tc.EmailAddress)
	assert.Equal(s.T(), phone, *tc.PhoneNumber)
	assert.Equal(s.T(), "Test", tc.FamilyName)

	err = srv.Delete(acctID)
	assert.Nil(s.T(), err)

	tc, err = srv.GetByID(acctID)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), tc)
}
