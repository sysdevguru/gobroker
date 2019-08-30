package corporateaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/suite"
)

type CorporateActionTestSuite struct {
	dbtest.Suite
	action *models.CorporateAction
}

func TestCorporateActionTestSuite(t *testing.T) {
	suite.Run(t, new(CorporateActionTestSuite))
}

func (s *CorporateActionTestSuite) SetupSuite() {
	s.SetupDB()
	s.action = &models.CorporateAction{
		AssetID: uuid.Must(uuid.NewV4()),
		Type:    enum.ReverseSplit,
		Date:    time.Now().In(calendar.NY).Format("2006-01-02"),
	}

	if err := db.DB().Create(s.action).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *CorporateActionTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *CorporateActionTestSuite) TestList() {
	srv := Service().WithTx(db.DB())

	actions, err := srv.List("")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), actions, 1)

	actions, err = srv.List(time.Now().In(calendar.NY).AddDate(0, 0, 1).Format("2006-01-02"))
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), actions)
}
