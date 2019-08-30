package apex

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/go-querystring/query"
)

var braggartPath = "/braggart/api/v1/transactions"

type TransactionType string

const (
	Allocation TransactionType = "ALLOCATION"
	Execution  TransactionType = "EXECUTION"
	Prime      TransactionType = "PRIME"
)

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type BraggartTransaction struct {
	Version     int    `json:"version"`
	ExternalID  string `json:"externalId"`
	Transaction struct {
		Type          TransactionType `json:"type"`
		AccountNumber string          `json:"accountNumber"`
		AccountType   string          `json:"accountType"`
		Side          struct {
			Type      Side   `json:"type"`
			ShortType string `json:"shortType,omitempty"`
		} `json:"side"`
		Quantity            int     `json:"quantity"`
		Price               float64 `json:"price"`
		Currency            string  `json:"currency"`
		TransactionDateTime string  `json:"transactionDateTime"`
		OpenClose           string  `json:"openClose,omitempty"`
		BrokerCapacity      string  `json:"brokerCapacity"`
		Route               struct {
			Type string `json:"type"`
		} `json:"route"`
		OrderID string `json:"orderId"`
	} `json:"transaction"`
	Instrument struct {
		Type         string `json:"type"`
		InstrumentID struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"instrumentId"`
	} `json:"instrument"`
}

type PostTransactionsResponse []PostTransactionsReceipt

type PostTransactionsReceipt struct {
	ID            *string `json:"id"`
	Correspondent *string `json:"correspondent"`
	ExternalID    *string `json:"externalId"`
	AccountNumber *string `json:"accountNumber"`
	Timestamp     *string `json:"timestamp"`
	Status        *string `json:"status"`
	Source        *struct {
		Type *string `json:"type"`
		Name *string `json:"name"`
	} `json:"source"`
	PastSources []struct {
		Type *string `json:"type"`
		Name *string `json:"name"`
	} `json:"pastSources"`
	ErrorDetails *string `json:"errorDetails"`
}

type GetTransactionResponse struct {
	Total *int `json:"total"`
	Data  []struct {
		ID            *string `json:"id"`
		Correspondent *string `json:"correspondent"`
		ExternalID    *string `json:"externalId"`
		AccountNumber *string `json:"accountNumber"`
		Timestamp     *string `json:"timestamp"`
		Status        *string `json:"status"`
		Source        *struct {
			Type *string `json:"type"`
			Name *string `json:"name"`
		} `json:"source"`
		PastSources []struct {
			Type *string `json:"type"`
			Name *string `json:"name"`
		} `json:"pastSources"`
		ErrorDetails *string `json:"errorDetails"`
	} `json:"data"`
}

func (a *Apex) PostTransactions(transactions []BraggartTransaction) (*PostTransactionsResponse, error) {
	if len(transactions) > 1000 {
		return nil, errors.New("only 1000 transactions per request")
	}
	uri := fmt.Sprintf(
		"%v%v",
		os.Getenv("APEX_URL"),
		braggartPath,
	)
	m := PostTransactionsResponse{}
	if _, err := a.call(uri, "POST", transactions, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) GetTransaction(id string) (*GetTransactionResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/%v",
		os.Getenv("APEX_URL"),
		braggartPath,
		id,
	)
	m := GetTransactionResponse{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

type BraggartTransactionQuery struct {
	Correspondent string   `url:"correspondent,omitempty"`
	Account       string   `url:"account,omitempty"`
	StartDate     string   `url:"startDate,omitempty"`
	EndDate       string   `url:"endDate,omitempty"`
	ExternalID    string   `url:"externalId,omitempty"`
	Source        string   `url:"source,omitempty"`
	SourceName    string   `url:"sourceName,omitempty"`
	Status        []string `url:"status,omitempty"`
	Limit         int      `url:"limit,omitempty"`
	Offset        int      `url:"offset,omitempty"`
}

type ListTransactionsResponse struct {
	Total *int                           `json:"total"`
	Data  []ListTransactionsResponseData `json:"data"`
}

type ListTransactionsResponseData struct {
	ID            *string `json:"id"`
	Correspondent *string `json:"correspondent"`
	ExternalID    *string `json:"externalId"`
	AccountNumber *string `json:"accountNumber"`
	Timestamp     *string `json:"timestamp"`
	Status        *string `json:"status"`
	Source        *struct {
		Type *string `json:"type"`
		Name *string `json:"name"`
	} `json:"source"`
	PastSources []struct {
		Type *string `json:"type"`
		Name *string `json:"name"`
	} `json:"pastSources"`
	ErrorDetails *string `json:"errorDetails"`
}

func (a *Apex) ListTransactions(q BraggartTransactionQuery) (ret *ListTransactionsResponse, err error) {
	v, _ := query.Values(q)
	uri := fmt.Sprintf(
		"%v%v?%v",
		os.Getenv("APEX_URL"),
		braggartPath,
		v.Encode(),
	)

	m := ListTransactionsResponse{}

	if _, err = a.getJSON(uri, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

func (a *Apex) RemoveTransaction(id string) error {
	uri := fmt.Sprintf(
		"%v%v/%v",
		os.Getenv("APEX_URL"),
		braggartPath,
		id,
	)
	if _, err := a.call(uri, "DELETE", nil, nil); err != nil {
		return err
	}
	return nil
}
