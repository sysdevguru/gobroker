package cognito

import (
	"bytes"
	"crypto/rsa"
	b64 "encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/alpacahq/gopaca/env"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
)

var (
	once    sync.Once
	enabled bool
	cli     *client
)

// Enabled returns whether the system is using cognito
// for auth, or bypassing for testing purposes
func Enabled() bool {
	once.Do(func() {
		enabled, _ = strconv.ParseBool(env.GetVar("COGNITO_ENABLED"))
		cli = &client{
			Region:      env.GetVar("COGNITO_REGION"),
			UserPoolID:  env.GetVar("COGNITO_USER_POOL_ID"),
			ClientID:    env.GetVar("COGNITO_CLIENT_ID"),
			RedirectURI: env.GetVar("COGNITO_REDIRECT_URI"),
		}
	})

	return enabled
}

// ListUsers returns the users in the Cognito identity pool
func ListUsers(attributes []*string) ([]*cognitoidentityprovider.UserType, error) {
	users := []*cognitoidentityprovider.UserType{}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := cognitoidentityprovider.New(sess, &aws.Config{
		Region: aws.String(cli.Region),
	})

	page := ""

	for {
		input := &cognitoidentityprovider.ListUsersInput{
			UserPoolId:      aws.String(cli.UserPoolID),
			AttributesToGet: attributes,
		}

		if page != "" {
			input.PaginationToken = aws.String(page)
		}

		listUsersOutput, err := svc.ListUsers(input)

		if err != nil {
			return nil, err
		}

		users = append(users, listUsersOutput.Users...)

		if len(listUsersOutput.Users) < 60 || listUsersOutput.PaginationToken == nil {
			break
		} else {
			page = *listUsersOutput.PaginationToken
		}
	}

	return users, nil
}

// ParseAndVerify a JWT token string with AWS Cognito
func ParseAndVerify(t string) (*jwt.Token, error) {
	if cli.WellKnownJWKs == nil {
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		svc := cognitoidentityprovider.New(sess, &aws.Config{
			Region: aws.String(cli.Region),
		})

		userPoolOutput, err := svc.DescribeUserPool(
			&cognitoidentityprovider.DescribeUserPoolInput{
				UserPoolId: aws.String(cli.UserPoolID),
			})

		if err == nil {
			cli.UserPoolType = userPoolOutput.UserPool
		} else {
			return nil, err
		}

		// Get the client app
		userPoolClientOutput, err := svc.DescribeUserPoolClient(&cognitoidentityprovider.DescribeUserPoolClientInput{
			ClientId:   aws.String(cli.ClientID),
			UserPoolId: aws.String(cli.UserPoolID),
		})
		if err == nil {
			cli.UserPoolClient = userPoolClientOutput.UserPoolClient
		} else {
			return nil, err
		}
		if cli.UserPoolClient != nil {
			// Set the Base64 <client_id>:<client_secret> for basic authorization header
			b := bytes.Buffer{}

			b.WriteString(cli.ClientID)
			b.WriteString(":")
			b.WriteString(aws.StringValue(cli.UserPoolClient.ClientSecret))
			base64AuthStr := b64.StdEncoding.EncodeToString(b.Bytes())
			b.Reset()

			b.WriteString("Basic ")
			b.WriteString(base64AuthStr)
			cli.Base64BasicAuthorization = b.String()
			b.Reset()

			// Set up login and signup URLs, if there is a domain available
			cli.getURLs()
		}

		// Set the well known JSON web token key sets
		if err = cli.getWellKnownJWTKs(); err != nil {
			return nil, err
		}
	}

	return cli.parseAndVerify(t)
}

type client struct {
	Region                   string
	UserPoolID               string
	ClientID                 string
	UserPoolType             *cognitoidentityprovider.UserPoolType
	UserPoolClient           *cognitoidentityprovider.UserPoolClientType
	WellKnownJWKs            *jwk.Set
	BaseURL                  string
	HostedLoginURL           string
	HostedLogoutURL          string
	HostedSignUpURL          string
	RedirectURI              string
	LogoutRedirectURI        string
	TokenEndpoint            string
	Base64BasicAuthorization string
}

func (c *client) getWellKnownJWTKs() error {
	// https://cognito-idp.<region>.amazonaws.com/<pool_id>/.well-known/jwks.json
	b := strings.Builder{}

	b.WriteString("https://cognito-idp.")
	b.WriteString(c.Region)
	b.WriteString(".amazonaws.com/")
	b.WriteString(c.UserPoolID)
	b.WriteString("/.well-known/jwks.json")
	wkjwksURL := b.String()
	b.Reset()

	// Use this cool package
	set, err := jwk.Fetch(wkjwksURL)
	if err == nil {
		c.WellKnownJWKs = set
	}

	return err
}

func (c *client) getURLs() {
	if c.UserPoolType != nil && c.UserPoolType.Domain != nil {
		// Get the base URL
		b := strings.Builder{}

		b.WriteString("https://")
		b.WriteString(aws.StringValue(c.UserPoolType.Domain))
		b.WriteString(".auth.")
		b.WriteString(c.Region)
		b.WriteString(".amazoncognito.com")
		c.BaseURL = b.String()
		b.Reset()

		// Set the HostedLoginURL
		b.WriteString(c.BaseURL)
		b.WriteString("/login?response_type=code&client_id=")
		b.WriteString(c.ClientID)
		b.WriteString("&redirect_uri=")
		b.WriteString(c.RedirectURI)
		c.HostedLoginURL = b.String()
		b.Reset()

		// Set the HostedLogoutURL
		b.WriteString(c.BaseURL)
		b.WriteString("/logout?response_type=code&client_id=")
		b.WriteString(c.ClientID)
		b.WriteString("&redirect_uri=")
		b.WriteString(c.RedirectURI)
		c.HostedLogoutURL = b.String()
		b.Reset()

		// Set the HostedSignUpURL
		b.WriteString(c.BaseURL)
		b.WriteString("/signup?response_type=code&client_id=")
		b.WriteString(c.ClientID)
		b.WriteString("&redirect_uri=")
		b.WriteString(c.RedirectURI)
		c.HostedSignUpURL = b.String()
		b.Reset()

		// Set the authorization token URL
		b.WriteString(c.BaseURL)
		b.WriteString("/oauth2/token")
		c.TokenEndpoint = b.String()
		b.Reset()
	}
}

func (c *client) parseAndVerify(t string) (*jwt.Token, error) {
	// 3 tokens are returned from the Cognito TOKEN endpoint; "id_token" "access_token" and "refresh_token"
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		// Looking up the key id will return an array of just one key
		keys := c.WellKnownJWKs.LookupKeyID(token.Header["kid"].(string))
		if len(keys) == 0 {
			return nil, fmt.Errorf("could not find matching `kid` in well known tokens")
		}
		// Build the public RSA key
		key, err := keys[0].Materialize()
		if err != nil {
			return nil, err
		}
		rsaPublicKey := key.(*rsa.PublicKey)
		return rsaPublicKey, nil
	})

	// Populated when you Parse/Verify a token
	// First verify the token itself is a valid format
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Then check time based claims; exp, iat, nbf
			err = claims.Valid()
			if err == nil {
				// Then check that `aud` matches the app client id
				// (if `aud` even exists on the token, second arg is a "required" option)
				if claims.VerifyAudience(c.ClientID, false) {
					return token, nil
				}
			}
		}
	}

	return nil, err
}
