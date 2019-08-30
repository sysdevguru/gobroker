package docrequest

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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DocRequestTestSuite struct {
	dbtest.Suite
	account *models.Account
}

func TestDocRequestTestSuite(t *testing.T) {
	suite.Run(t, new(DocRequestTestSuite))
}

func (s *DocRequestTestSuite) SetupSuite() {
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

	legalName := "Joe Trader"

	details := &models.OwnerDetails{
		OwnerID:   s.account.Owners[0].ID,
		LegalName: &legalName,
	}

	if err := db.DB().Create(details).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *DocRequestTestSuite) TearDownSuite() {
	s.TeardownDB()
}

type TestSnapClient struct {
	GetSnapResponse  *string
	PostSnapResponse *string
}

func (t TestSnapClient) GetSnap(snapID string) (*string, error) {
	return t.GetSnapResponse, nil
}

func (t TestSnapClient) PostSnap(image []byte, name string, tag apex.SnapTag) (*string, error) {
	return t.PostSnapResponse, nil
}

func (s *DocRequestTestSuite) TestUpload() {
	tx := db.DB()
	srv := docRequestService{tx: tx}
	assert.Nil(s.T(), tx.Exec("truncate table snaps").Error)
	assert.Nil(s.T(), tx.Exec("truncate table investigations").Error)
	assert.Nil(s.T(), tx.Exec("truncate table document_requests").Error)

	mcli := TestSnapClient{}
	bytes := "some random image bytes"
	mockResp := "snapId"
	mcli.GetSnapResponse = &bytes
	mcli.PostSnapResponse = &mockResp

	// Case - no investigation

	err := srv.Upload(s.account.IDAsUUID(), []byte(bytes), models.DriverLicense, models.Back, "image/jpeg", mcli)
	assert.NotNil(s.T(), err)

	// Case - investigation is already submitted

	inv := models.Investigation{
		ID:        uuid.Must(uuid.NewV4()).String(),
		AccountID: s.account.ID,
		Status:    models.SketchPending,
	}

	srv.tx.Create(&inv)

	err = srv.Upload(s.account.IDAsUUID(), []byte(bytes), models.DriverLicense, models.Back, "image/jpeg", mcli)
	assert.NotNil(s.T(), err)

	// Case - does not matched with requested document types
	inv.Status = models.SketchRejected
	srv.tx.Save(&inv)

	err = srv.Upload(s.account.IDAsUUID(), []byte(bytes), models.DriverLicense, models.Back, "image/jpeg", mcli)
	assert.NotNil(s.T(), err)

	// Case - Success
	doc := models.DocumentRequest{
		InvestigationID: inv.ID,
		AccountID:       inv.AccountID,
		Status:          models.DocumentRequestRequested,
		DocumentType:    models.DriverLicense,
	}
	srv.tx.Create(&doc)

	err = srv.Upload(s.account.IDAsUUID(), []byte(bytes), models.DriverLicense, models.Back, "image/jpeg", mcli)
	assert.Nil(s.T(), err)
	srv.tx.Where("id = ?", doc.ID).Preload("Snaps").First(&doc)

	assert.Equal(s.T(), doc.Status, models.DocumentRequestRequested)
	assert.Equal(s.T(), doc.Snaps[0].ID, "snapId")

	mockResp = "snapId2"
	mcli.PostSnapResponse = &mockResp
	err = srv.Upload(s.account.IDAsUUID(), []byte(bytes), models.DriverLicense, models.Front, "image/jpeg", mcli)
	assert.Nil(s.T(), err)
	srv.tx.Where("id = ?", doc.ID).First(&doc)
	assert.Equal(s.T(), doc.Status, models.DocumentRequestUploaded)
}

func (s *DocRequestTestSuite) TestList() {
	mcli := TestSnapClient{}
	bytes := "some random image bytes"
	mockResp := "snapId"
	mcli.GetSnapResponse = &bytes
	mcli.PostSnapResponse = &mockResp

	tx := db.DB()
	srv := docRequestService{tx: tx}

	id := s.account.IDAsUUID()
	docs, err := srv.List(&id, nil, mcli)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), docs, 0)

	// Case - with investigation

	inv := models.Investigation{
		ID:        uuid.Must(uuid.NewV4()).String(),
		AccountID: s.account.ID,
		Status:    models.SketchRejected,
	}

	srv.tx.Create(&inv)

	docs, err = srv.List(&id, nil, mcli)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), docs, 0)

	// Case - with investigation and documents

	categories := []models.DocumentCategory{models.UPIC}

	err = srv.Request(inv.ID, categories)
	assert.Nil(s.T(), err)

	docs, err = srv.List(&id, nil, mcli)

	assert.Nil(s.T(), err)
	require.Len(s.T(), docs, 3)

	// Case - with investigation and snap
	docID := docs[0]["id"].(string)

	snap := models.Snap{
		ID:                "snapId",
		AccountID:         s.account.ID,
		DocumentRequestID: docID,
		MimeType:          "image/jpeg",
		Name:              "PASSPORT",
	}
	err = srv.tx.Create(&snap).Error
	assert.Nil(s.T(), err)

	docs, err = srv.List(&id, nil, mcli)

	assert.Nil(s.T(), err)
	assert.Len(s.T(), docs, 3)
	assert.Equal(s.T(), docs[0]["id"], docID)

	snaps := docs[0]["snaps"].([]map[string]interface{})
	assert.Equal(s.T(), snaps[0]["id"], "snapId")
	assert.Equal(s.T(), snaps[0]["mime_type"], "image/jpeg")

	// Case - for last document requests as well
	docs, err = srv.ListLast(s.account.IDAsUUID(), mcli)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), docs, 3)
}
