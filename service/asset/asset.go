package asset

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type AssetService interface {
	GetByID(assetID uuid.UUID) (*models.Asset, error)
	List(class *enum.AssetClass, status *enum.AssetStatus) ([]*models.Asset, error)
	WithTx(tx *gorm.DB) AssetService
}

type assetService struct {
	AssetService
	tx *gorm.DB
}

func Service() AssetService {
	return &assetService{}
}

func (s *assetService) WithTx(tx *gorm.DB) AssetService {
	s.tx = tx
	return s
}

func (a *assetService) GetByID(assetID uuid.UUID) (*models.Asset, error) {
	asset := &models.Asset{}

	q := a.tx.Where("id = ?", assetID).Find(asset)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("asset not found for %v", assetID))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(a.tx.Error)
	}

	return asset, nil
}

func (a *assetService) List(class *enum.AssetClass, status *enum.AssetStatus) ([]*models.Asset, error) {
	assets := []*models.Asset{}

	q := a.tx

	if class == nil {
		q = q.Where("class = ?", enum.AssetClassUSEquity)
	} else {
		q = q.Where("class = ?", *class)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}

	q = q.Find(&assets)

	if q.RecordNotFound() {
		return assets, nil
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return assets, nil
}
