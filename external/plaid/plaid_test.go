package plaid

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alpacahq/gopaca/env"
	"github.com/stretchr/testify/suite"
	"github.com/valyala/fasthttp"
)

type PlaidTestSuite struct {
	suite.Suite
}

func TestPlaidTestSuite(t *testing.T) {
	suite.Run(t, new(PlaidTestSuite))
}

func (s *PlaidTestSuite) SetupSuite() {
	pc = &PlaidClient{}
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		assert.FailNow(s.T(), "unmocked request!")
		return nil
	}
	env.RegisterDefault("PLAID_SECRET", "some random secret")
	env.RegisterDefault("PLAID_CLIENT_ID", "test_client")
	env.RegisterDefault("PLAID_URL", "https://plaid.base.url")
	env.RegisterDefault("PLAID_PUBLIC_KEY", "test_key")
}
