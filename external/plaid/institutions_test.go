package plaid

import (
	"encoding/json"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func (s *PlaidTestSuite) TestListInstitutions() {
	// clean request
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		body, _ := json.Marshal(map[string]interface{}{
			"institutions": "some institution data",
		})
		resp.SetBody(body)
		resp.SetStatusCode(200)
		return nil
	}
	inst, _ := pc.ListInstitutions()
	assert.NotNil(s.T(), inst)

	// >400
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(400)
		return fmt.Errorf("400 code")
	}
	inst, _ = pc.ListInstitutions()
	assert.Nil(s.T(), inst)
}

func (s *PlaidTestSuite) TestGetInstitution() {
	// clean request
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		body, _ := json.Marshal(map[string]interface{}{
			"institution": "some institution data",
		})
		resp.SetBody(body)
		resp.SetStatusCode(200)
		return nil
	}
	inst, _ := pc.GetInstitution("ins_0")
	assert.NotNil(s.T(), inst)

	// >400
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(400)
		return fmt.Errorf("400 code")
	}
	inst, _ = pc.GetInstitution("ins_0")
	assert.Nil(s.T(), inst)
}
