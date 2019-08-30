package apiclient

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/profitloss"
	"github.com/gofrs/uuid"
	"github.com/valyala/fasthttp"
)

type InternalClient struct {
	*RestClient
	email string
	token string
}

type AccessKey struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	Secret    string     `json:"secret"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func NewInternalClient(email, password string, id *string) *InternalClient {
	c := &InternalClient{
		RestClient: NewRestClient("", ""),
		email:      email,
	}

	if id != nil {
		c.token = *id
	}

	c.RestClient.setHeaderFunc = c.setHeader

	return c
}

func (c *InternalClient) CreateAccount() (res *Account, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts", c.RestClient.baseUrl)
	params := map[string]interface{}{
		"email": c.email,
	}
	_, err = c.call(url, "POST", params, &res)

	if res != nil {
		c.token = res.ID
	}

	return
}

func (c *InternalClient) CreateAccessKey(accountID string) (res *AccessKey, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/access_keys", c.RestClient.baseUrl)
	params := map[string]interface{}{
		"account_id": accountID,
	}
	_, err = c.call(url, "POST", params, &res)
	return
}

func (c *InternalClient) ListAccessKeys() (res []*AccessKey, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/access_keys", c.RestClient.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *InternalClient) DeleteAccessKey(keyID string) (err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/access_keys/%s", c.RestClient.baseUrl, keyID)
	_, err = c.call(url, "DELETE", nil, nil)
	return
}

func (c *InternalClient) GetOwnerDetails(accountID string) (res *models.OwnerDetails, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/details", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *InternalClient) PatchOwnerDetails(accountID string, params map[string]interface{}) (err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/details", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "PATCH", params, nil)
	return
}

func (c *InternalClient) GetAffiliates(accountID string) (res []*models.Affiliate, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/affiliates", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *InternalClient) CreateAffiliate(accountID string, params map[string]interface{}) (res *models.Affiliate, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/affiliates", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "POST", params, &res)
	return
}

func (c *InternalClient) PatchAffiliate(accountID string, affiliateID uint, params map[string]interface{}) (res *models.Affiliate, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/affiliates/%v", c.RestClient.baseUrl, accountID, affiliateID)
	_, err = c.call(url, "PATCH", params, &res)
	return
}

func (c *InternalClient) DeleteAffiliate(accountID string, affiliateID uint) (err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/affiliates/%v", c.RestClient.baseUrl, accountID, affiliateID)
	_, err = c.call(url, "DELETE", nil, nil)
	return
}

func (c *InternalClient) GetTrustedContact(accountID string) (res *models.TrustedContact, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/trusted_contact", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *InternalClient) CreateTrustedContact(accountID string, params map[string]interface{}) (res *models.TrustedContact, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/trusted_contact", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "POST", params, &res)
	return
}

func (c *InternalClient) PatchTrustedContact(accountID string, params map[string]interface{}) (res *models.TrustedContact, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/trusted_contact", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "PATCH", params, &res)
	return
}

func (c *InternalClient) DeleteTrustedContact(accountID string) (err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/trusted_contact", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "DELETE", nil, nil)
	return
}

func (c *InternalClient) GetProfitLoss(accountID string) (res *profitloss.ProfitLoss, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/profitloss", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *InternalClient) CreateTransfer(accountID string, req entities.TransferRequest) (res *models.Transfer, err error) {
	url := fmt.Sprintf("%v/api/_internal/v1/accounts/%v/transfers", c.RestClient.baseUrl, accountID)
	_, err = c.call(url, "POST", req, &res)
	return
}

func (c *InternalClient) setHeader(req *fasthttp.Request) {
	if c.token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	} else {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", uuid.Must(uuid.NewV4()).String()))
	}
}
