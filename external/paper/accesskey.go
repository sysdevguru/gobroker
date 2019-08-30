package paper

import (
	"fmt"
	"net/http"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/gofrs/uuid"
)

// again workaround struct being used here for now until PT & GB are unified
func (c *Client) CreateAccessKey(accountID uuid.UUID) (key entities.AccessKeyEntity, err error) {

	err = c.request(
		http.MethodPost,
		fmt.Sprintf("/_internal/v1/accounts/%v/access_keys", accountID), map[string]interface{}{}, &key)

	return key, err
}

func (c *Client) DeleteAccessKey(accountID uuid.UUID, keyID string) (err error) {

	err = c.request(
		http.MethodDelete,
		fmt.Sprintf("/_internal/v1/accounts/%v/access_keys/%v", accountID, keyID), nil, nil)

	return err
}

func (c *Client) ListAccessKeys(accountID uuid.UUID) (keys []models.AccessKey, err error) {
	err = c.request(
		http.MethodGet,
		fmt.Sprintf("/_internal/v1/accounts/%v/access_keys", accountID), nil, &keys)
	return keys, err
}
