package corporateaction

import (
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/jinzhu/gorm"
)

type CorporateActionService interface {
	List(date string) ([]models.CorporateAction, error)
	WithTx(tx *gorm.DB) CorporateActionService
}

type corporateActionService struct {
	CorporateActionService
	tx *gorm.DB
}

func Service() CorporateActionService {
	return &corporateActionService{}
}

func (s *corporateActionService) WithTx(tx *gorm.DB) CorporateActionService {
	s.tx = tx
	return s
}

func (s *corporateActionService) List(date string) ([]models.CorporateAction, error) {
	actions := []models.CorporateAction{}

	q := s.tx

	if date == "" {
		date = time.Now().In(calendar.NY).Format("2006-01-02")
	}

	q = q.Where("date = ?", date).Find(&actions)

	if q.Error != nil && q.Error != gorm.ErrRecordNotFound {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	if len(actions) == 0 {
		return actions, nil
	}

	return actions, nil
}
