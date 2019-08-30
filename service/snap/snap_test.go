package snap

import (
	"testing"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SnapTestSuite struct {
	dbtest.Suite
	account *models.Account
	inv     *models.Investigation
	docReq  *models.DocumentRequest
}

func TestSnapTestSuite(t *testing.T) {
	suite.Run(t, new(SnapTestSuite))
}

func (s *SnapTestSuite) SetupSuite() {
	s.SetupDB()
	amt, _ := decimal.NewFromString("10000")
	apexAcct := "apca_test"
	s.account = &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
			},
		},
	}
	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.inv = &models.Investigation{
		ID:        uuid.Must(uuid.NewV4()).String(),
		AccountID: uuid.Must(uuid.NewV4()).String(),
		Status:    models.SketchIndeterminate,
	}
	if err := db.DB().Create(s.inv).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.docReq = &models.DocumentRequest{
		AccountID:       s.account.ID,
		InvestigationID: s.inv.ID,
		DocumentType:    models.DriverLicense,
		Status:          models.DocumentRequestUploaded,
	}
	if err := db.DB().Create(s.docReq).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *SnapTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *SnapTestSuite) TestSnap() {
	srv := snapService{
		tx: db.DB(),
		postFunc: func(image []byte, name string, tag apex.SnapTag) (*string, error) {
			id := uuid.Must(uuid.NewV4()).String()
			return &id, nil
		},
		getFunc: func(id string) (*string, error) {
			preview := uuid.Must(uuid.NewV4()).String()
			return &preview, nil
		},
	}

	snaps, err := srv.List(s.account.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), snaps)

	image := []byte("some_image_bytes")
	snap, err := srv.Upload(s.account.IDAsUUID(), image, s.docReq, "image/png")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), snap)
	assert.Equal(s.T(), s.account.ID, snap.AccountID)

	snaps, err = srv.List(s.account.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Len(s.T(), snaps, 1)
	assert.Equal(s.T(), snap.ID, snaps[0].ID)
}
