package snap

import (
	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type SnapService interface {
	List(id uuid.UUID) ([]models.Snap, error)
	Upload(id uuid.UUID, image []byte, doc *models.DocumentRequest, mimeType string) (*models.Snap, error)
	WithTx(tx *gorm.DB) SnapService
}

type snapService struct {
	SnapService
	tx       *gorm.DB
	postFunc func(image []byte, name string, tag apex.SnapTag) (*string, error)
	getFunc  func(id string) (*string, error)
}

func (s *snapService) WithTx(tx *gorm.DB) SnapService {
	s.tx = tx
	return s
}

func Service() SnapService {
	return &snapService{
		postFunc: apex.Client().PostSnap,
		getFunc:  apex.Client().GetSnap,
	}
}

func (s *snapService) List(id uuid.UUID) ([]models.Snap, error) {
	snaps := []models.Snap{}

	q := s.tx.Where("account_id = ?", id).Preload("DocumentRequest").Order("created_at DESC").Find(&snaps)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	for i := range snaps {
		preview, err := s.getFunc(snaps[i].ID)
		if err != nil {
			return nil, gberrors.InternalServerError.WithMsg("failed to retrieve snap from apex")
		}

		snaps[i].Preview = preview
	}

	return snaps, nil
}

func (s *snapService) Upload(id uuid.UUID, image []byte, doc *models.DocumentRequest, mimeType string) (*models.Snap, error) {
	snapID, err := s.postFunc(image, doc.DocumentType.String(), apex.ID_DOCUMENT)
	if err != nil {
		return nil, err
	}
	snap := &models.Snap{
		ID:                *snapID,
		AccountID:         id.String(),
		MimeType:          mimeType,
		Name:              doc.DocumentType.String(),
		DocumentRequestID: doc.ID,
	}
	if err := s.tx.Create(snap).Error; err != nil {
		return nil, err
	}
	return snap, nil
}
