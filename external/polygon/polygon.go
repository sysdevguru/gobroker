package polygon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/env"
	"github.com/valyala/fasthttp"
)

type ListSymbolsResponse struct {
	Symbols []struct {
		Symbol          string `json:"symbol"`
		Name            string `json:"name"`
		Type            string `json:"type"`
		Updated         string `json:"updated"`
		IsOTC           bool   `json:"isOTC"`
		PrimaryExchange int    `json:"primaryExchange"`
		ExchSym         string `json:"exchSym"`
		URL             string `json:"url"`
	} `json:"symbols"`
}

func ListSymbols() (*ListSymbolsResponse, error) {
	resp := ListSymbolsResponse{}
	page := 0

	for {
		url := fmt.Sprintf("%v/v1/meta/symbols?apiKey=%s&sort=%s&perpage=%v&page=%v",
			env.GetVar("POLYGON_URL"),
			env.GetVar("POLYGON_API_TOKEN"),
			"symbol", 200, page)

		code, body, err := fasthttp.Get(nil, url)
		if err != nil {
			return nil, err
		}

		if code >= fasthttp.StatusMultipleChoices {
			return nil, fmt.Errorf("status code %v", code)
		}

		r := &ListSymbolsResponse{}

		err = json.Unmarshal(body, r)

		if err != nil {
			return nil, err
		}

		if len(r.Symbols) == 0 {
			break
		}

		resp.Symbols = append(resp.Symbols, r.Symbols...)

		page++
	}

	return &resp, nil
}

type ListExchangesResponse []struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Market string `json:"market"`
	Mic    string `json:"mic"`
	Name   string `json:"name"`
	Tape   string `json:"tape"`
}

func ListExchanges() (*ListExchangesResponse, error) {
	url := fmt.Sprintf("%v/v1/meta/exchanges?apiKey=%s",
		env.GetVar("POLYGON_URL"),
		env.GetVar("POLYGON_API_TOKEN"))

	code, body, err := fasthttp.Get(nil, url)
	if err != nil {
		return nil, err
	}

	if code >= fasthttp.StatusMultipleChoices {
		return nil, fmt.Errorf("status code %v", code)
	}

	r := &ListExchangesResponse{}

	err = json.Unmarshal(body, r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

type AgreementType string

const (
	NYSE   AgreementType = "nyse"
	NASDAQ AgreementType = "nasdaq"
)

type NyseBody struct {
	Name       string     `json:"name"`
	Address    string     `json:"address"`
	Occupation Occupation `json:"occupation"`
	Questions  []bool     `json:"questions"`
}

type Occupation struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Title     string `json:"title"`
	Functions string `json:"functions"`
}

func NewNyseBody(acct *models.Account) (NyseBody, error) {
	var occupation Occupation
	if acct.PrimaryOwner().Details.EmploymentStatus != nil &&
		*acct.PrimaryOwner().Details.EmploymentStatus == models.Employed {

		occupation = Occupation{
			Name:      *acct.PrimaryOwner().Details.Position,
			Address:   *acct.PrimaryOwner().Details.EmployerAddress,
			Title:     *acct.PrimaryOwner().Details.Position,
			Functions: *acct.PrimaryOwner().Details.Function,
		}
	} else {
		name := "N/A"
		if acct.PrimaryOwner().Details.EmploymentStatus != nil {
			name = string(*acct.PrimaryOwner().Details.EmploymentStatus)
		}

		occupation = Occupation{
			Name:      name,
			Address:   "N/A",
			Title:     "N/A",
			Functions: "N/A",
		}
	}

	addr, err := acct.PrimaryOwner().Details.FormatAddress()
	if err != nil {
		return NyseBody{}, err
	}

	return NyseBody{
		Name:       *acct.PrimaryOwner().Details.LegalName,
		Address:    addr,
		Occupation: occupation,
		Questions: []bool{
			true,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
			false,
		},
	}, nil
}

type NasdaqBody struct {
	Name      string `json:"name"`
	Date      string `json:"date"`
	Pro       int    `json:"pro"`
	Signature int    `json:"signature"`
}

func NewNasdaqBody(acct *models.Account) *NasdaqBody {
	return &NasdaqBody{
		Name:      *acct.PrimaryOwner().Details.LegalName,
		Date:      time.Now().In(calendar.NY).Format("1-2-2006"),
		Pro:       0,
		Signature: 1,
	}
}

func AgreementBody(acct *models.Account, name string) (body interface{}, err error) {
	switch name {
	case "nyse":
		body, err = NewNyseBody(acct)
	case "nasdaq":
		body = NewNasdaqBody(acct)
	default:
		err = gberrors.InvalidRequestParam.WithMsg("invalid agreement type")
	}

	return
}

func Agreement(name string, agreement interface{}) ([]byte, error) {
	url := fmt.Sprintf("%v/v1/agreements/%s?apiKey=%s",
		env.GetVar("POLYGON_URL"),
		name,
		env.GetVar("POLYGON_API_TOKEN"))

	buf, err := json.Marshal(agreement)

	if err != nil {
		return nil, err
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	req.SetBody(buf)
	req.Header.SetMethod(http.MethodPost)
	req.Header.Set("Content-Type", "application/json")
	resp := fasthttp.AcquireResponse()

	if err = fasthttp.Do(req, resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() > fasthttp.StatusMultipleChoices {
		return nil, fmt.Errorf("status code %v (%v)", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Body(), nil
}
