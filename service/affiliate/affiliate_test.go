package affiliate

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AffiliateTestSuite struct {
	dbtest.Suite
	affiliate *models.Affiliate
}

func TestAffiliateTestSuite(t *testing.T) {
	suite.Run(t, new(AffiliateTestSuite))
}

func (s *AffiliateTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *AffiliateTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AffiliateTestSuite) TestAffiliate() {
	srv := affiliateService{tx: db.DB()}

	acctID := uuid.Must(uuid.NewV4())
	affiliates, err := srv.List(acctID)
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), affiliates)

	aff := &models.Affiliate{
		AccountID:     acctID.String(),
		StreetAddress: address.Address{"123 Somewhere Ln"},
		City:          "Somewhere",
		State:         "SW",
		PostalCode:    "12345",
		Country:       "USA",
		CompanyName:   "Evil Corp.",
	}

	affiliate, err := srv.Create(aff)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), affiliate)
	assert.Equal(s.T(), aff.AccountID, affiliate.AccountID)
	assert.Equal(s.T(), aff.CompanyName, affiliate.CompanyName)

	aff = &models.Affiliate{
		AccountID:     acctID.String(),
		StreetAddress: address.Address{"123 Nowhere Blvd"},
		City:          "Nowhere",
		State:         "NW",
		PostalCode:    "54321",
		Country:       "USA",
		CompanyName:   "Good Corp.",
	}
	aff, err = srv.Create(aff)
	assert.Nil(s.T(), err)

	affiliates, err = srv.List(acctID)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), affiliates, 2)

	err = srv.Delete(acctID, aff.ID)
	assert.Nil(s.T(), err)

	affiliates, err = srv.List(acctID)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), affiliates, 1)

	patches := map[string]interface{}{
		"street_address": "123 Patched Street",
		"city":           aff.City,
		"state":          aff.State,
		"postal_code":    aff.PostalCode,
		"country":        aff.Country,
		"company_name":   aff.CompanyName,
	}

	aff, err = srv.Patch(acctID, affiliates[0].ID, patches)
	require.Nil(s.T(), err)
	assert.NotNil(s.T(), aff)
	assert.Equal(s.T(), patches["city"].(string), aff.City)
	assert.Equal(s.T(), patches["company_name"], aff.CompanyName)
}
