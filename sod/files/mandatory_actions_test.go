package files

import (
	"io/ioutil"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/sod/files/samples"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *FileTestSuite) TestMandatoryActions() {
	{
		asset := &models.Asset{
			Symbol:    "UGLD",
			Class:     enum.AssetClassUSEquity,
			Exchange:  "NASDAQ",
			CUSIP:     "22542D688",
			Status:    enum.AssetActive,
			Tradable:  true,
			Shortable: true,
		}

		require.Nil(s.T(), db.DB().Create(asset).Error)

		position := &models.Position{
			AssetID:      asset.IDAsUUID(),
			AccountID:    uuid.Must(uuid.NewV4()).String(),
			Status:       models.Open,
			EntryOrderID: uuid.Must(uuid.NewV4()).String(),
			Side:         models.Long,
			EntryPrice:   decimal.New(100, 0),
		}

		require.Nil(s.T(), db.DB().Create(position).Error)

		f, err := samples.SamplesBundle.Open("samples/EXT235_3AP_20181012.TXT")
		require.Nil(s.T(), err)
		require.NotNil(s.T(), f)

		buf, err := ioutil.ReadAll(f)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), buf)

		sodFile := &MandatoryActionReport{}

		assert.Nil(s.T(), Parse(buf, sodFile))
		assert.NotPanics(s.T(), func() { sodFile.Sync(s.asOf) })

		p := &models.Position{}

		require.Nil(s.T(), db.DB().Where("id = ?", position.ID).First(p).Error)
		assert.NotNil(s.T(), p.MarkedForSplitAt)

		a := &models.Asset{}
		require.Nil(s.T(), db.DB().Where("id = ?", asset.ID).First(a).Error)
		assert.Equal(s.T(), "22542D316", a.CUSIP)
	}

	{
		asset := &models.Asset{
			Symbol:    "YECO",
			Class:     enum.AssetClassUSEquity,
			Exchange:  "NASDAQ",
			CUSIP:     "G98847208",
			Status:    enum.AssetActive,
			Tradable:  true,
			Shortable: true,
		}

		require.Nil(s.T(), db.DB().Create(asset).Error)

		// create cusip_old = ""
		asset2 := &models.Asset{
			Symbol:    "ZZZ",
			Class:     enum.AssetClassUSEquity,
			Exchange:  "NASDAQ",
			CUSIP:     "012345",
			SymbolOld: "",
			CUSIPOld:  "",
			Status:    enum.AssetActive,
			Tradable:  true,
			Shortable: true,
		}
		require.Nil(s.T(), db.DB().Create(asset2).Error)

		limitPrice := decimal.New(100, 0)
		order := &models.Order{
			AssetID:     asset.ID,
			Account:     "3AP090000",
			Symbol:      "YECO",
			Qty:         decimal.New(1, 0),
			Side:        enum.Buy,
			TimeInForce: enum.GTC,
			Type:        enum.Limit,
			LimitPrice:  &limitPrice,
			Status:      enum.OrderNew,
		}
		require.Nil(s.T(), db.DB().Create(order).Error)

		f, err := samples.SamplesBundle.Open("samples/EXT235_3AP_20181214.TXT")
		require.Nil(s.T(), err)
		require.NotNil(s.T(), f)

		buf, err := ioutil.ReadAll(f)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), buf)

		sodFile := &MandatoryActionReport{}

		assert.Nil(s.T(), Parse(buf, sodFile))
		assert.NotPanics(s.T(), func() { sodFile.Sync(s.asOf) })

		a := &models.Asset{}
		require.Nil(s.T(), db.DB().Where("id = ?", asset.ID).First(a).Error)
		assert.Equal(s.T(), "YECOF", a.Symbol)
		assert.Equal(s.T(), "YECO", a.SymbolOld)

		a2 := &models.Asset{}
		require.Nil(s.T(), db.DB().Where("id = ?", asset2.ID).First(a2).Error)
		// make sure nothing changes
		assert.Equal(s.T(), "ZZZ", a2.Symbol)
		assert.Equal(s.T(), "", a2.SymbolOld)
		assert.Equal(s.T(), "012345", a2.CUSIP)
		assert.Equal(s.T(), "", a2.CUSIPOld)
	}
}
