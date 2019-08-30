package agreements

import (
	"bytes"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/external/polygon"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gobroker/utils"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type AgreementsService interface {
	Get(accountID uuid.UUID, t polygon.AgreementType) ([]byte, error)
	Accept(accountID uuid.UUID, t polygon.AgreementType) error
	WithTx(tx *gorm.DB) AgreementsService
}

type agreementsService struct {
	AgreementsService
	tx                   *gorm.DB
	polygonAgreementFunc func(name string, body interface{}) ([]byte, error)
	agreementStorageFunc func(name string, data []byte) error
}

func (s *agreementsService) WithTx(tx *gorm.DB) AgreementsService {
	s.tx = tx
	return s
}

func Service() AgreementsService {
	return &agreementsService{
		polygonAgreementFunc: polygon.Agreement,
		agreementStorageFunc: func(fileName string, data []byte) (err error) {
			return s3man.New().Upload(bytes.NewReader(data), fmt.Sprintf("/agreements/%s", fileName))
		},
	}
}

func (s *agreementsService) Get(accountID uuid.UUID, t polygon.AgreementType) ([]byte, error) {
	acct, err := account.Service().WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return nil, err
	}

	// fill out the doc
	body, err := polygon.AgreementBody(acct, string(t))
	if err != nil {
		return nil, err
	}

	// generate the doc w/ polygon
	pdf, err := s.polygonAgreementFunc(string(t), body)
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return pdf, nil
}

func (s *agreementsService) Accept(accountID uuid.UUID, t polygon.AgreementType) error {
	acct, err := account.Service().WithTx(s.tx).GetByID(accountID)
	if err != nil {
		return err
	}

	// fill out the doc
	body, err := polygon.AgreementBody(acct, string(t))
	if err != nil {
		return gberrors.InvalidRequestParam.WithMsg(err.Error())
	}

	// generate the doc w/ polygon
	pdf, err := s.polygonAgreementFunc(string(t), body)
	if err != nil {
		return gberrors.InternalServerError.WithError(err)
	}

	// mark agreement signed
	patch, err := getSignedPatch(t)
	if err != nil {
		return gberrors.InvalidRequestParam.WithMsg(err.Error())
	}

	if _, err = ownerdetails.Service().WithTx(s.tx).Patch(acct.IDAsUUID(), patch); err != nil {
		return err
	}

	// store the doc to s3 (if not in DEV)
	if !utils.Dev() {
		if err = s.agreementStorageFunc(fmt.Sprintf("%s/%s.pdf", t, acct.ID), pdf); err != nil {
			return gberrors.InternalServerError.WithError(err)
		}
	}

	return nil
}

func getSignedPatch(t polygon.AgreementType) (map[string]interface{}, error) {
	switch t {
	case polygon.NASDAQ:
		return map[string]interface{}{"nasdaq_agreement_signed_at": time.Now()}, nil
	case polygon.NYSE:
		return map[string]interface{}{"nyse_agreement_signed_at": time.Now()}, nil
	default:
		return nil, fmt.Errorf("invalid agreement")
	}
}
