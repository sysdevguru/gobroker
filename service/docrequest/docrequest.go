package docrequest

import (
	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type DocRequestService interface {
	List(accountID *uuid.UUID, investigationID *string, client SnapClient) ([]map[string]interface{}, error)
	ListLast(accountID uuid.UUID, client SnapClient) ([]map[string]interface{}, error)
	Upload(accountID uuid.UUID, image []byte, documentType models.DocumentType, documentSubType models.DocumentSubType, mimeType string, client SnapClient) error
	Request(investigationID string, docCategories []models.DocumentCategory) error
	WithTx(tx *gorm.DB) DocRequestService
}

type docRequestService struct {
	DocRequestService
	tx *gorm.DB
}

func Service() DocRequestService {
	return &docRequestService{}
}

func (s *docRequestService) WithTx(tx *gorm.DB) DocRequestService {
	s.tx = tx
	return s
}

type SnapClient interface {
	PostSnap([]byte, string, apex.SnapTag) (*string, error)
	GetSnap(string) (*string, error)
}

func documentRequest2Map(doc models.DocumentRequest, client SnapClient) (map[string]interface{}, error) {

	mdoc := map[string]interface{}{
		"id":               doc.ID,
		"investigation_id": doc.InvestigationID,
		"account_id":       doc.AccountID,
		"status":           doc.Status,
		"document_type":    doc.DocumentType,
		"created_at":       doc.CreatedAt,
		"updated_at":       doc.UpdatedAt,
	}

	snaps := []map[string]interface{}{}

	for i := range doc.Snaps {
		s := map[string]interface{}{}
		s["id"] = doc.Snaps[i].ID
		preview, err := client.GetSnap(doc.Snaps[i].ID)
		if err != nil {
			return nil, err
		}
		s["preview"] = preview
		s["ale_confirmed_at"] = doc.Snaps[i].ALEConfirmedAt
		s["mime_type"] = doc.Snaps[i].MimeType
		snaps = append(snaps, s)
	}

	mdoc["snaps"] = snaps

	return mdoc, nil
}

func (s *docRequestService) ListLast(accountID uuid.UUID, client SnapClient) ([]map[string]interface{}, error) {
	inv := models.Investigation{}
	documents := []models.DocumentRequest{}
	out := []map[string]interface{}{}

	q := s.tx.Where("account_id = ?", accountID.String()).Order("created_at DESC").First(&inv)

	if q.RecordNotFound() {
		return out, nil
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	if !(inv.Status == models.SketchIndeterminate || inv.Status == models.SketchRejected) {
		return out, nil
	}

	if err := s.tx.
		Where("investigation_id = ?", inv.ID).
		Preload("Snaps", "deleted_at IS NULL").
		Order("created_at DESC").
		Find(&documents).Error; err != nil {

		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	for i := range documents {
		m, err := documentRequest2Map(documents[i], client)
		if err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}

		out = append(out, m)
	}

	return out, nil
}

func (s *docRequestService) List(accountID *uuid.UUID, investigationID *string, client SnapClient) ([]map[string]interface{}, error) {
	documents := []models.DocumentRequest{}
	q := s.tx

	if accountID != nil {
		q = q.Where("account_id = ?", accountID.String())
	}

	if investigationID != nil {
		q = q.Where("investigation_id = ?", *investigationID)
	}

	q = q.Preload("Snaps", "deleted_at IS NULL").Order("created_at DESC").Find(&documents)

	if q.RecordNotFound() {
		return []map[string]interface{}{}, nil
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	out := []map[string]interface{}{}

	for i := range documents {
		m, err := documentRequest2Map(documents[i], client)
		if err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}

		out = append(out, m)
	}

	return out, nil
}

func (s *docRequestService) Upload(accountID uuid.UUID, image []byte, documentType models.DocumentType, documentSubType models.DocumentSubType, mimeType string, client SnapClient) error {
	inv := models.Investigation{}
	document := models.DocumentRequest{}

	acct, err := account.Service().WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return err
	}

	q := s.tx.Where("account_id = ?", accountID.String()).Order("created_at DESC").First(&inv)

	if q.RecordNotFound() {
		return gberrors.NotFound.WithMsg("investigation not found")
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	q = q.Where("investigation_id = ?", inv.ID).
		Where("document_type = ?", documentType.String()).
		Preload("Snaps", "deleted_at IS NULL").
		First(&document)

	if q.RecordNotFound() {
		return gberrors.NotFound.WithMsg("document request not found")
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	snapID, err := client.PostSnap(image, documentSubType.String(), apex.ID_DOCUMENT)
	if err != nil {
		return err
	}

	if err := s.tx.
		Where("investigation_id = ?", inv.ID).
		Where("document_type = ?", documentType.String()).
		Preload("Snaps", "deleted_at IS NULL").
		Set("gorm:query_option", db.ForUpdate).
		First(&document).Error; err != nil {
		return err
	}

	for i := range document.Snaps {
		if document.Snaps[i].Name == documentSubType.String() {
			t := clock.Now()
			document.Snaps[i].DeletedAt = &t
			if err := s.tx.Save(document.Snaps[i]).Error; err != nil {
				return err
			}
			break
		}
	}

	snap := &models.Snap{
		ID:                *snapID,
		AccountID:         accountID.String(),
		DocumentRequestID: document.ID,
		MimeType:          mimeType,
		Name:              documentSubType.String(),
	}

	if err := s.tx.Create(snap).Error; err != nil {
		return gberrors.InternalServerError.WithError(err)
	}

	if document.IsComplete(snap) {
		// Need to select status not to update associations.
		if err := s.tx.Model(&document).Select("status").Update("status", models.DocumentRequestUploaded).Error; err != nil {
			return err
		}

		// notify via slack
		{
			msg := slack.NewAccountUpdate()
			msg.SetBody(struct {
				Type         string `json:"type"`
				DocumentType string `json:"document_type"`
				ApexAccount  string `json:"apex_account"`
				Name         string `json:"name"`
				Email        string `json:"email"`
			}{
				"documents_uploaded",
				string(documentType),
				*acct.ApexAccount,
				*acct.PrimaryOwner().Details.LegalName,
				acct.PrimaryOwner().Email,
			})

			slack.Notify(msg)
		}

	}

	return nil
}

func (s *docRequestService) Request(investigationID string, docCategories []models.DocumentCategory) error {
	inv := models.Investigation{}

	q := s.tx.Where("id = ?", investigationID).Find(&inv)

	if q.RecordNotFound() {
		return gberrors.NotFound.WithMsg("investigation not found")
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	docRequests := []models.DocumentRequest{}

	if err := q.Where("investigation_id = ? ", inv.ID).Find(&docRequests).Error; err != nil {
		return gberrors.InternalServerError.WithError(err)
	}

	// make this operation idempotent, only adding the missing
	// document requests based on the new set of categories
	for _, docCat := range docCategories {
		for _, docType := range docCat.Types() {

		CreateMissing:
			for _, existingReq := range docRequests {
				if existingReq.DocumentType == docType {
					break CreateMissing
				}
			}

			newDocReq := models.DocumentRequest{
				InvestigationID: inv.ID,
				AccountID:       inv.AccountID,
				Status:          models.DocumentRequestRequested,
				DocumentType:    docType,
			}

			if err := s.tx.Create(&newDocReq).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
