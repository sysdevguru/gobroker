package apex

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alpacahq/apex/encryption"
	"github.com/valyala/fasthttp"
)

const (
	confirmsQuery    = "?argCustomerID=%v&argCustomerOfficeID=&argDateTime=%v&argUUID=&argFirm=%v&argAccountNumber=%v&argBeginDate=%v&argEndDate=%v"
	docServicesQuery = "?argCustomerID=%v&argCustomerOfficeID=&argDateTime=%v&argFirm=%v&argAccountNumber=%v&argBeginMonthMM=%02d&argBeginYearYYYY=%v&argEndMonthMM=%02d&argEndYearYYYY=%v&argDocumentType=%v"
)

type PensonResponseCustomerConfirmGetListResponse struct {
	XMLName      xml.Name `xml:"PensonResponseCustomerConfirmGetList"`
	Text         string   `xml:",chardata"`
	Xsi          string   `xml:"xsi,attr"`
	Xsd          string   `xml:"xsd,attr"`
	Xmlns        string   `xml:"xmlns,attr"`
	PensonStatus struct {
		Text       string `xml:",chardata"`
		StatusCode struct {
			Text string `xml:",chardata"`
		} `xml:"StatusCode"`
		StatusDateTime struct {
			Text string `xml:",chardata"`
		} `xml:"StatusDateTime"`
	} `xml:"PensonStatus"`
	PensonData struct {
		Text                          string `xml:",chardata"`
		CustomerConfirmGetListDataSet struct {
			Text                          string `xml:",chardata"`
			CustomerConfirmGetListDataRow []struct {
				Text        string `xml:",chardata"`
				ProcessDate struct {
					Text string `xml:",chardata"`
				} `xml:"ProcessDate"`
				Subject struct {
					Text string `xml:",chardata"`
				} `xml:"Subject"`
				Status struct {
					Text string `xml:",chardata"`
				} `xml:"Status"`
				ConfirmURL struct {
					Text string `xml:",chardata"`
				} `xml:"ConfirmURL"`
			} `xml:"CustomerConfirmGetListDataRow"`
		} `xml:"CustomerConfirmGetListDataSet"`
	} `xml:"PensonData"`
}

func (a *Apex) getAccountConfirms(id string, start, end time.Time) ([]Document, error) {
	ts, err := encryption.EncryptedTimestamp()
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(
		confirmsQuery,
		os.Getenv("APEX_CUSTOMER_ID"),
		*ts,
		os.Getenv("APEX_FIRM_CODE"),
		id,
		start.Format("01/02/2006"),
		end.Format("01/02/2006"),
	)

	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("text/xml")
	req.SetRequestURI(fmt.Sprintf(
		"%v/%v%v",
		os.Getenv("APEX_WS_URL"),
		"/CustomerConfirm.asmx/GetListUpdated",
		q,
	))

	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()

	if err = a.request(req, resp, time.Minute); err != nil {
		return nil, err
	}

	pResp := PensonResponseCustomerConfirmGetListResponse{}

	if err := xml.Unmarshal(resp.Body(), &pResp); err != nil {
		return nil, fmt.Errorf("account confirm xml unmarshal failure (%v)", err)
	}

	rows := pResp.PensonData.CustomerConfirmGetListDataSet.CustomerConfirmGetListDataRow

	ret := make([]Document, len(rows))
	for i, row := range rows {
		ret[i] = Document{
			Account: id,
			Date:    row.ProcessDate.Text,
			URL:     row.ConfirmURL.Text,
			Type:    TradeConfirmation,
		}
	}

	return ret, nil
}

type PensonResponseCGIDocumentGetDirectListResponse struct {
	XMLName      xml.Name `xml:"PensonResponseCGIDocumentGetDirectList"`
	Text         string   `xml:",chardata"`
	Xsi          string   `xml:"xsi,attr"`
	Xsd          string   `xml:"xsd,attr"`
	Xmlns        string   `xml:"xmlns,attr"`
	PensonStatus struct {
		Text       string `xml:",chardata"`
		StatusCode struct {
			Text string `xml:",chardata"`
		} `xml:"StatusCode"`
		StatusDescription struct {
			Text string `xml:",chardata"`
		} `xml:"StatusDescription"`
		StatusDateTime struct {
			Text string `xml:",chardata"`
		} `xml:"StatusDateTime"`
	} `xml:"PensonStatus"`
	PensonData struct {
		Text               string `xml:",chardata"`
		CGIDocumentDataSet struct {
			Text               string `xml:",chardata"`
			CGIDocumentDataRow []struct {
				Text  string `xml:",chardata"`
				DocID struct {
					Text string `xml:",chardata"`
				} `xml:"DocID"`
				AccountNumber struct {
					Text string `xml:",chardata"`
				} `xml:"AccountNumber"`
				DocDate struct {
					Text string `xml:",chardata"`
				} `xml:"DocDate"`
				RepID struct {
					Text string `xml:",chardata"`
				} `xml:"RepID"`
				AccountName struct {
					Text string `xml:",chardata"`
				} `xml:"AccountName"`
				DocType struct {
					Text string `xml:",chardata"`
				} `xml:"DocType"`
				URL struct {
					Text string `xml:",chardata"`
				} `xml:"URL"`
				Insert1 struct {
					Text string `xml:",chardata"`
				} `xml:"Insert1"`
				Insert2 struct {
					Text string `xml:",chardata"`
				} `xml:"Insert2"`
				PageCount struct {
					Text string `xml:",chardata"`
				} `xml:"PageCount"`
				PageCountDisplay struct {
					Text string `xml:",chardata"`
				} `xml:"PageCountDisplay"`
			} `xml:"CGIDocumentDataRow"`
		} `xml:"CGIDocumentDataSet"`
	} `xml:"PensonData"`
}

type DocumentType int

func (t *DocumentType) String() string {
	switch *t {
	case TradeConfirmation:
		return "trade_confirmation"
	case AccountStatement:
		return "account_statement"
	case Tax1099:
		return "tax_1099"
	case Tax1099R:
		return "tax_1099r"
	case Tax1042S:
		return "tax_1042s"
	case Tax5498:
		return "tax_5498"
	case Tax5498ESA:
		return "tax_5498esa"
	case TaxFMV:
		return "tax_fmv"
	case Tax1099Q:
		return "tax_1099q"
	case TaxSDIRA:
		return "tax_sdira"
	default:
		return "all"
	}
}

func DocumentTypeFromString(str string) DocumentType {
	switch str {
	case "trade_confirmation":
		return TradeConfirmation
	case "account_statement":
		return AccountStatement
	case "tax_1099":
		return Tax1099
	case "tax_1099r":
		return Tax1099R
	case "tax_1042s":
		return Tax1042S
	case "tax_5498":
		return Tax5498
	case "tax_5498esa":
		return Tax5498ESA
	case "tax_fmv":
		return TaxFMV
	case "tax_1099q":
		return Tax1099Q
	case "tax_sdira":
		return TaxSDIRA
	default:
		return All
	}
}

const (
	// made-up by us to enable the unified interface
	All               DocumentType = -1
	TradeConfirmation DocumentType = 0
	// specified by apex
	AccountStatement DocumentType = 2
	Tax1099          DocumentType = 100
	Tax1099R         DocumentType = 102
	Tax1042S         DocumentType = 104
	Tax5498          DocumentType = 105
	Tax5498ESA       DocumentType = 106
	TaxFMV           DocumentType = 107
	Tax1099Q         DocumentType = 108
	TaxSDIRA         DocumentType = 109
)

type Document struct {
	URL     string
	Account string
	Date    string
	Type    DocumentType
}

func (d *Document) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		URL     string `json:"url"`
		Account string `json:"account"`
		Date    string `json:"date"`
		Type    string `json:"type"`
	}{
		URL:     d.URL,
		Account: d.Account,
		Date:    d.Date,
		Type:    d.Type.String(),
	})
}

func (a *Apex) docService(id string, start, end time.Time, docType string) ([]Document, error) {
	ts, err := encryption.EncryptedTimestamp()
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(
		docServicesQuery,
		os.Getenv("APEX_CUSTOMER_ID"),
		*ts,
		os.Getenv("APEX_FIRM_CODE"),
		id,
		start.Month(),
		start.Year(),
		end.Month(),
		end.Year(),
		docType,
	)

	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("text/xml")
	req.SetRequestURI(fmt.Sprintf(
		"%v/%v%v",
		os.Getenv("APEX_WS_URL"),
		"CGIDocument.asmx/GetDirectListUpdated",
		q,
	))

	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()
	if err = a.request(req, resp, time.Minute); err != nil {
		return nil, err
	}

	pResp := PensonResponseCGIDocumentGetDirectListResponse{}

	if err := xml.Unmarshal(resp.Body(), &pResp); err != nil {
		return nil, fmt.Errorf("account confirm xml unmarshal failure (%s)", err)
	}

	rows := pResp.PensonData.CGIDocumentDataSet.CGIDocumentDataRow

	ret := make([]Document, len(rows))
	for i, row := range rows {

		t, err := strconv.ParseInt(row.DocType.Text, 10, 64)
		if err != nil {
			return nil, err
		}

		ret[i] = Document{
			Account: id,
			Date:    row.DocDate.Text,
			URL:     row.URL.Text,
			Type:    DocumentType(t),
		}
	}

	return ret, nil
}

// GetDocuments retrieves documents from Apex associated with the
// specified account number.
func (a *Apex) GetDocuments(account string, start, end time.Time, docType DocumentType) ([]Document, error) {
	switch docType {
	case All:
		confirms, err := a.getAccountConfirms(account, start, end)
		if err != nil {
			return nil, err
		}

		srvDocs, err := a.docService(account, start, end, "")
		if err != nil {
			return nil, err
		}

		return append(confirms, srvDocs...), nil
	case TradeConfirmation:
		return a.getAccountConfirms(account, start, end)
	default:
		return a.docService(account, start, end, strconv.FormatInt(int64(docType), 10))
	}
}
