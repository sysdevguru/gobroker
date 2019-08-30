package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/alpacahq/gopaca/env"
	"github.com/valyala/fasthttp"
)

func init() {
	// alpacadev credentials
	env.RegisterDefault("EGNYTE_KEY", "vb7v2bgp7u2t5pkam2jmjj3x")
	env.RegisterDefault("EGNYTE_USERNAME", "devreg-admin")
	env.RegisterDefault("EGNYTE_PASSWORD", "JAqeUFRYn=8Vu*3t")
	env.RegisterDefault("EGNYTE_DOMAIN", "alpacadev.egnyte.com")
}

func main() {
	token, err := GetToken()
	if err != nil {
		panic(err)
	}

	fmt.Println("Token: ", *token)
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func GetToken() (*string, error) {
	u, err := url.Parse("https://" + path.Join(env.GetVar("EGNYTE_DOMAIN"), "puboauth", "token"))
	if err != nil {
		return nil, err
	}

	q := u.Query()

	q.Set("grant_type", "password")
	q.Set("client_id", env.GetVar("EGNYTE_KEY"))
	q.Set("username", env.GetVar("EGNYTE_USERNAME"))
	q.Set("password", env.GetVar("EGNYTE_PASSWORD"))

	u.RawQuery = q.Encode()

	// build request
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(u.String())
	req.Header.SetContentType("application/x-www-form-urlencoded")
	req.Header.SetMethod(http.MethodPost)

	resp := fasthttp.AcquireResponse()

	if err = fasthttp.Do(req, resp); err != nil {
		return nil, err
	}

	if resp.StatusCode() > fasthttp.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"egnyte authentication failed (code: %v body: %v)",
			resp.StatusCode(), string(resp.Body()))
	}

	auth := AuthResponse{}

	if err = json.Unmarshal(resp.Body(), &auth); err != nil {
		return nil, err
	}

	return &auth.AccessToken, nil
}
