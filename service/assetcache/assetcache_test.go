package assetcache

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
)

var dummyUUID, _ = uuid.NewV1()

type TestSuite struct {
	suite.Suite
}

func TestRunSuite(t *testing.T) {
	loadAssets = loadAssetsMock
	suite.Run(t, new(TestSuite))
}

func loadAssetsMock() ([]*models.Asset, error) {
	return []*models.Asset{
		&models.Asset{
			ID:       "asset-1",
			Class:    "us_equity",
			Exchange: "NASDAQ",
			Symbol:   "AAPL",
			Status:   enum.AssetActive,
			Tradable: true,
		},
		&models.Asset{
			ID:       dummyUUID.String(),
			Class:    "us_equity",
			Exchange: "NYSE",
			Symbol:   "BAC",
			Status:   enum.AssetInactive,
			Tradable: false,
		},
	}, nil
}

func (s *TestSuite) TestAssetCache() {
	c := GetAssetCache()

	var a *models.Asset

	a = c.Get("asset-1")
	assert.Equal(s.T(), "AAPL", a.Symbol)

	a = c.Get("fake-id")
	assert.Nil(s.T(), a)

	a = c.Get("BAC")
	assert.False(s.T(), a.Tradable)

	a = c.Get("AAPL:NASDAQ")
	assert.Equal(s.T(), "AAPL", a.Symbol)

	a = c.Get("AAPL:NASDAQ:us_equity")
	assert.Equal(s.T(), a.Symbol, "AAPL")

	a = c.Get("X:CBOE")
	assert.Nil(s.T(), a)

	a = Get("AAPL")
	assert.Equal(s.T(), "AAPL", a.Symbol)

	a = GetByID(dummyUUID)
	assert.Equal(s.T(), "BAC", a.Symbol)

	id, _ := uuid.NewV4()
	a = GetByID(id)
	assert.Nil(s.T(), a)
}
