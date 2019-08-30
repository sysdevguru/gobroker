package models

import (
	"testing"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AuthSuite struct {
	suite.Suite
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}

func (s *AuthSuite) TestNewAccessKey() {
	id, _ := uuid.NewV4()
	key, _ := models.NewAccessKey(id, enum.LiveAccount)
	assert.Nil(s.T(), key.Verify(key.Secret))
	assert.NotNil(s.T(), key.Verify("invalid"))
}
