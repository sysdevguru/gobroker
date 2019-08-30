package txlevel

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gopaca/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TxLevelSuite struct {
	dbtest.Suite
}

func TestTxLevelSuitee(t *testing.T) {
	suite.Run(t, new(TxLevelSuite))
}

func (s *TxLevelSuite) SetupSuite() {
	s.SetupDB()
}

func (s *TxLevelSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *TxLevelSuite) TestRepeatable() {
	{
		tx := db.DB()
		ok, err := Repeatable(tx)
		assert.Nil(s.T(), err)
		assert.False(s.T(), ok)
	}

	{
		tx := db.RepeatableRead()
		ok, err := Repeatable(tx)
		assert.Nil(s.T(), err)
		assert.True(s.T(), ok)
		if err := tx.Commit().Error; err != nil {
			assert.FailNow(s.T(), err.Error())
		}
	}

	{
		tx := db.Serializable()
		ok, err := Repeatable(tx)
		assert.Nil(s.T(), err)
		assert.True(s.T(), ok)
		if err := tx.Commit().Error; err != nil {
			assert.FailNow(s.T(), err.Error())
		}
	}

}
