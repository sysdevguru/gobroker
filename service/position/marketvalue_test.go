package position

import (
	"math/big"

	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gopaca/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (s *PositionTestSuite) TestMarketValueAt() {
	srv := Service(assetcache.GetAssetCache()).WithTx(db.DB())

	{
		value, err := srv.MarketValueAt(s.accountID, s.date)
		assert.Nil(s.T(), err)

		qty := decimal.NewFromBigInt(big.NewInt(int64(100)), 0)
		price := decimal.NewFromFloat(109)
		assert.True(s.T(), value.Equals(qty.Mul(price)))
	}

	{
		value, err := srv.MarketValueAt(s.accountID, s.date.Prev())
		assert.Nil(s.T(), err)
		assert.True(s.T(), value.Equals(decimal.Zero))
	}
}
