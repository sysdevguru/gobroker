package mailer

import (
	"testing"

	"github.com/alpacahq/gobroker/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MailerTestSuite struct {
	suite.Suite
	account    *models.Account
	marginCall *models.MarginCall
}

func TestMailerTestSuite(t *testing.T) {
	suite.Run(t, new(MailerTestSuite))
}

func (s *MailerTestSuite) TestMaskedAccount() {
	masked := MaskApexAccount("3AP1234")
	assert.Equal(s.T(), masked, "3AP...4")

	masked = MaskApexAccount("3AP12345")
	assert.Equal(s.T(), masked, "3AP...45")

	masked = MaskApexAccount("3AP123456")
	assert.Equal(s.T(), masked, "3AP...456")

	masked = MaskApexAccount("3AP1234567")
	assert.Equal(s.T(), masked, "3AP....567")
}
