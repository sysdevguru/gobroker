package fundamental

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FundamentalTestSuite struct {
	dbtest.Suite
	asset *models.Asset
	fund  *models.Fundamental
}

func TestFundamentalTestSuite(t *testing.T) {
	suite.Run(t, new(FundamentalTestSuite))
}

func (s *FundamentalTestSuite) SetupSuite() {
	s.SetupDB()
	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}
	if err := db.DB().Create(s.asset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.fund = &models.Fundamental{
		AssetID:  s.asset.ID,
		Symbol:   s.asset.Symbol,
		FullName: "Apple Inc.",
	}
	if err := db.DB().Create(s.fund).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *FundamentalTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *FundamentalTestSuite) TestGetByID() {
	srv := Service().WithTx(db.DB())

	f, err := srv.GetByID(s.asset.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), f)

	f, err = srv.GetByID(uuid.Must(uuid.NewV4()))
	assert.Nil(s.T(), f)
	assert.NotNil(s.T(), err)
}

func (s *FundamentalTestSuite) TestGetByIDs() {
	srv := Service().WithTx(db.DB())

	f, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), f)
	assert.Len(s.T(), f, 1)

	f, err = srv.GetByIDs([]uuid.UUID{uuid.Must(uuid.NewV4())})
	assert.NotNil(s.T(), f)
	assert.Len(s.T(), f, 0)
	assert.Nil(s.T(), err)
}
