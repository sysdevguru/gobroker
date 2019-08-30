package egnyte

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/alpacahq/gopaca/env"
	"github.com/valyala/fasthttp"
)

// Upload a zip file to egnyte
func Upload(filePath string, data []byte) error {
	u, err := url.Parse(
		"https://" + path.Join(
			env.GetVar("EGNYTE_DOMAIN"),
			"pubapi/v1/fs-content/Shared",
			filePath,
		))

	if err != nil {
		return err
	}

	// build request
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(u.String())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", env.GetVar("EGNYTE_TOKEN")))
	req.Header.SetContentType("application/zip")
	req.Header.SetMethod(http.MethodPost)
	req.SetBody(data)

	resp := fasthttp.AcquireResponse()

	if err = fasthttp.Do(req, resp); err != nil {
		return err
	}

	if resp.StatusCode() > fasthttp.StatusMultipleChoices {
		return fmt.Errorf(
			"egnyte upload failed (response: %v)",
			resp.String())
	}

	return nil
}

// CreateDirectory creates a folder in egnyte
func CreateDirectory(dirPath string) error {
	u, err := url.Parse(
		"https://" + path.Join(
			env.GetVar("EGNYTE_DOMAIN"),
			"pubapi/v1/fs/Shared",
			dirPath,
		))

	if err != nil {
		return err
	}

	buf, _ := json.Marshal(map[string]interface{}{"action": "add_folder"})

	// build request
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(u.String())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", env.GetVar("EGNYTE_TOKEN")))
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(buf)

	resp := fasthttp.AcquireResponse()

	if err = fasthttp.Do(req, resp); err != nil {
		return err
	}

	if resp.StatusCode() > fasthttp.StatusMultipleChoices {
		return fmt.Errorf(
			"egnyte folder creation failed (response: %v)",
			resp.String())
	}

	return nil
}
