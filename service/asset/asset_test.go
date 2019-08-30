package asset

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

type AssetTestSuite struct {
	dbtest.Suite
	activeAsset   *models.Asset
	inactiveAsset *models.Asset
}

func TestAssetTestSuite(t *testing.T) {
	suite.Run(t, new(AssetTestSuite))
}

func (s *AssetTestSuite) SetupSuite() {
	s.SetupDB()
	s.activeAsset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}
	s.inactiveAsset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "LNKD",
		Status:   enum.AssetInactive,
		Tradable: false,
	}
	if err := db.DB().Create(s.activeAsset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err := db.DB().Create(s.inactiveAsset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *AssetTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AssetTestSuite) TestGetByID() {
	srv := Service().WithTx(db.DB())

	a, err := srv.GetByID(s.activeAsset.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), a.ID, s.activeAsset.ID)

	a, err = srv.GetByID(uuid.Must(uuid.NewV4()))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), a)
}

func (s *AssetTestSuite) TestList() {
	srv := Service().WithTx(db.DB())

	equity := enum.AssetClassUSEquity
	active := enum.AssetActive
	assets, err := srv.List(&equity, &active)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), assets, 1)

	assets, err = srv.List(&equity, nil)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), assets, 2)

	assets, err = srv.List(nil, nil)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), assets, 2)
}
