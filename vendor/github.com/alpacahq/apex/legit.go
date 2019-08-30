package apex

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/square/go-jose.v2"
)

var legitPath = "/legit/api/v1/"

func jws() string {
	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.HS512,
			Key:       []byte(os.Getenv("APEX_SECRET")),
		},
		nil)
	if err != nil {
		panic(err)
	}
	payload, err := json.Marshal(
		map[string]interface{}{
			"username": os.Getenv("APEX_USER"),
			"entity":   os.Getenv("APEX_ENTITY"),
			"datetime": time.Now().Format(time.RFC3339),
		})
	if err != nil {
		panic(err)
	}
	object, err := signer.Sign(payload)
	if err != nil {
		panic(err)
	}
	h, err := json.Marshal(map[string]interface{}{"alg": "HS512"})
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(
		"%v.%v.%v",
		strings.Replace(b64.URLEncoding.EncodeToString(h), "=", "", -1),
		strings.Replace(b64.URLEncoding.EncodeToString(payload), "=", "", -1),
		strings.Replace(b64.URLEncoding.EncodeToString(object.Signatures[0].Signature), "=", "", -1),
	)
}

// Authenticate generates the JWT token w/ apex's
// Legit API
func (a *Apex) Authenticate() (err error) {
	if a.Dev {
		return nil
	}

	s := jws()
	uri := fmt.Sprintf(
		"%v%vcc/token?jws=%v",
		os.Getenv("APEX_URL"),
		legitPath,
		s,
	)

	var body []byte

	if _, err = a.call(uri, "GET", nil, &body); err != nil {
		return
	}

	jwt := string(body)

	if len(jwt) > 0 {
		a.JWT = jwt
		return nil
	}

	return errors.New("empty JWT returned by legit")
}
