package files

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/stretchr/testify/suite"
)

type FileTestSuite struct {
	dbtest.Suite
	asOf time.Time
}

func TestFileTestSuite(t *testing.T) {
	suite.Run(t, new(FileTestSuite))
}

func (s *FileTestSuite) SetupSuite() {
	s.asOf = time.Date(2018, 4, 17, 0, 0, 0, 0, calendar.NY)
	s.SetupDB()
}

func (s *FileTestSuite) TearDownSuite() {
	s.TeardownDB()
}
