package trade

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/stretchr/testify/suite"
)

type TradeWorkerTestSuite struct {
	dbtest.Suite
	asset *models.Asset
}

func TestTradeWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(TradeWorkerTestSuite))
}

func (s *TradeWorkerTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *TradeWorkerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *TradeWorkerTestSuite) TestTradeWorker() {

}
