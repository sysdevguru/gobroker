package paper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

func (c *Client) PolygonAuth(apiKeyID string) (uuid.UUID, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI(c.polygonPaperURL("/auth"))

	authReq := struct {
		APIKeyID string `json:"api_key_id"`
	}{
		APIKeyID: apiKeyID,
	}

	buf, err := json.Marshal(authReq)
	if err != nil {
		return uuid.Nil, err
	}

	req.SetBody(buf)

	resp := fasthttp.AcquireResponse()

	if err = c.requestFunc(req, resp); err != nil {
		return uuid.Nil, errors.Wrap(err, "failed request func")
	}

	if resp.StatusCode() > 300 {
		var apiError APIError

		if err := json.Unmarshal(resp.Body(), &apiError); err != nil {
			return uuid.Nil, errors.Wrap(err, fmt.Sprintf("failed to parse error (status_code = %v)", resp.StatusCode()))
		}
		apiError.StatusCode = resp.StatusCode()

		return uuid.Nil, &apiError
	}

	sub := map[string]uuid.UUID{}

	if err = json.Unmarshal(resp.Body(), &sub); err != nil {
		return uuid.Nil, err
	}

	return sub["user_id"], nil
}

func (c *Client) PolygonList(apiKeyIDs []string) (map[string]uuid.UUID, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI(c.polygonPaperURL("/auth"))

	subReq := struct {
		APIKeyIDs []string `json:"api_key_ids"`
	}{
		APIKeyIDs: apiKeyIDs,
	}

	buf, err := json.Marshal(subReq)
	if err != nil {
		return nil, err
	}

	req.SetBody(buf)

	resp := fasthttp.AcquireResponse()

	if resp.StatusCode() == http.StatusNotFound {
		return map[string]uuid.UUID{}, nil
	}

	if resp.StatusCode() > 300 {
		var apiError APIError

		if err := json.Unmarshal(resp.Body(), &apiError); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse error (status_code = %v)", resp.StatusCode()))
		}
		apiError.StatusCode = resp.StatusCode()

		return nil, &apiError
	}

	sub := map[string]uuid.UUID{}

	if err = json.Unmarshal(resp.Body(), &sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (c *Client) polygonPaperURL(endpoint string) string {
	u, _ := url.Parse(c.baseURL)
	u.Path = path.Join(u.Path, "/_polygon/v1/", endpoint)

	return u.String()
}
