package quote

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type QuoteService interface {
	GetByID(assetID uuid.UUID) (*QuoteAsAsset, error)
	GetByIDs(assetIDs []uuid.UUID) ([]*QuoteAsAsset, error)
	WithTx(tx *gorm.DB) QuoteService
}

type quoteService struct {
	QuoteService
	tx         *gorm.DB
	getQuotes  func(symbols []string) ([]price.Quote, error)
	assetcache assetcache.AssetCache
}

func Service(assetcache assetcache.AssetCache) QuoteService {
	return &quoteService{getQuotes: price.Quotes, assetcache: assetcache}
}

type QuoteAsAsset struct {
	price.Quote
	AssetID uuid.UUID       `json:"asset_id"`
	Symbol  string          `json:"symbol"`
	Class   enum.AssetClass `json:"asset_class"`
}

func (s *quoteService) WithTx(tx *gorm.DB) QuoteService {
	s.tx = tx
	return s
}

func (s *quoteService) GetByIDs(assetIDs []uuid.UUID) ([]*QuoteAsAsset, error) {
	if len(assetIDs) == 0 {
		return []*QuoteAsAsset{}, nil
	}

	var assets []models.Asset
	q := s.tx.Where("id IN (?)", assetIDs).Find(&assets)

	if len(assets) == 0 {
		return nil, gberrors.NotFound
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	qa := []*QuoteAsAsset{}

	symbols := make([]string, len(assets))

	for i, asset := range assets {
		symbols[i] = asset.Symbol
	}

	quotes, err := s.getQuotes(symbols)
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	for i, asset := range assets {
		qa = append(qa, &QuoteAsAsset{
			Quote:   quotes[i],
			AssetID: asset.IDAsUUID(),
			Symbol:  asset.Symbol,
			Class:   asset.Class,
		})
	}

	return qa, nil
}

func (s *quoteService) GetByID(assetID uuid.UUID) (*QuoteAsAsset, error) {
	asset := s.assetcache.GetByID(assetID)
	if asset == nil {
		return nil, gberrors.NotFound
	}

	quotes, err := s.getQuotes([]string{asset.Symbol})
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	qasset := QuoteAsAsset{
		Quote:   quotes[0],
		AssetID: assetID,
		Symbol:  asset.Symbol,
		Class:   asset.Class,
	}

	return &qasset, nil
}
