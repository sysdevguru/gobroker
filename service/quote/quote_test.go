package quote

import (
	"fmt"
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type QuoteTestSuite struct {
	dbtest.Suite
	asset *models.Asset
}

func TestQuoteTestSuite(t *testing.T) {
	suite.Run(t, new(QuoteTestSuite))
}

func (s *QuoteTestSuite) SetupSuite() {
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
}

func (s *QuoteTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *QuoteTestSuite) TestGetByID() {
	srv := quoteService{
		tx: db.DB(),
		getQuotes: func(symbols []string) ([]price.Quote, error) {
			quotes := make([]price.Quote, len(symbols))
			for i := range symbols {
				quotes[i] = price.Quote{
					BidTimestamp:  clock.Now(),
					Bid:           100.32,
					AskTimestamp:  clock.Now(),
					Ask:           100.32,
					LastTimestamp: clock.Now(),
					Last:          100.32,
				}
			}
			return quotes, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	quote, err := srv.GetByID(s.asset.IDAsUUID())
	assert.NotNil(s.T(), quote)
	assert.Nil(s.T(), err)

	quote, err = srv.GetByID(uuid.Must(uuid.NewV4()))
	assert.Nil(s.T(), quote)
	assert.NotNil(s.T(), err)

	srv.getQuotes = func(symbols []string) ([]price.Quote, error) {
		return nil, fmt.Errorf("no quote!")
	}
	quote, err = srv.GetByID(s.asset.IDAsUUID())
	assert.Nil(s.T(), quote)
	assert.NotNil(s.T(), err)
}

func (s *QuoteTestSuite) TestGetByIDs() {
	srv := quoteService{
		tx: db.DB(),
		getQuotes: func(symbols []string) ([]price.Quote, error) {
			quotes := make([]price.Quote, len(symbols))
			for i := range symbols {
				quotes[i] = price.Quote{
					BidTimestamp:  clock.Now(),
					Bid:           100.32,
					AskTimestamp:  clock.Now(),
					Ask:           100.32,
					LastTimestamp: clock.Now(),
					Last:          100.32,
				}
			}
			return quotes, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	quotes, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()})
	assert.NotNil(s.T(), quotes)
	assert.Len(s.T(), quotes, 1)
	assert.Nil(s.T(), err)

	quotes, err = srv.GetByIDs([]uuid.UUID{uuid.Must(uuid.NewV4())})
	assert.Nil(s.T(), quotes)
	assert.NotNil(s.T(), err)

	srv.getQuotes = func(symbols []string) ([]price.Quote, error) {
		return nil, fmt.Errorf("no quote!")
	}
	quotes, err = srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()})
	assert.Nil(s.T(), quotes)
	assert.Len(s.T(), quotes, 0)
	assert.NotNil(s.T(), err)
}
