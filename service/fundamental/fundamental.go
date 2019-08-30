package fundamental

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type FundamentalService interface {
	GetByID(assetID uuid.UUID) (*models.Fundamental, error)
	GetByIDs(assetIDs []uuid.UUID) ([]*models.Fundamental, error)
	WithTx(tx *gorm.DB) FundamentalService
}

type fundamentalService struct {
	FundamentalService
	tx *gorm.DB
}

func Service() FundamentalService {
	return &fundamentalService{}
}

func (s *fundamentalService) WithTx(tx *gorm.DB) FundamentalService {
	s.tx = tx
	return s
}

func (s *fundamentalService) GetByID(assetID uuid.UUID) (*models.Fundamental, error) {
	fundamental := &models.Fundamental{}

	q := s.tx.Where("asset_id = ?", assetID).Find(fundamental)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("asset not found for %v", assetID))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(s.tx.Error)
	}

	return fundamental, nil
}

func (s *fundamentalService) GetByIDs(assetIDs []uuid.UUID) ([]*models.Fundamental, error) {
	fundamentals := []*models.Fundamental{}

	q := s.tx.Where("asset_id IN (?)", assetIDs).Find(&fundamentals)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return fundamentals, nil
}
