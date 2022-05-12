package wallet

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/lbryio/lbrytv/apps/lbrytv/config"
	"github.com/lbryio/lbrytv/models"
	"github.com/volatiletech/null"

	"golang.org/x/oauth2"
)

const (
	TestClientID = "ci-tester"

	envClientSecret = "OAUTH_TEST_CLIENT_SECRET"
	envUsername     = "OAUTH_TEST_USERNAME"
	envPassword     = "OAUTH_TEST_PASSWORD"

	msgMissingEnv = "test oauth client env var %s is not set"
)

type TestAnyAuthenticator struct{}

func (a *TestAnyAuthenticator) Authenticate(token, ip string) (*models.User, error) {
	return &models.User{ID: 994, IdpID: null.StringFrom("my-idp-id")}, nil
}

func (a *TestAnyAuthenticator) GetTokenFromRequest(r *http.Request) (string, error) {
	return "", nil
}

type TestMissingTokenAuthenticator struct{}

func (a *TestMissingTokenAuthenticator) Authenticate(token, ip string) (*models.User, error) {
	return nil, nil
}

func (a *TestMissingTokenAuthenticator) GetTokenFromRequest(r *http.Request) (string, error) {
	return "", ErrNoAuthInfo
}

// GetTestToken is for easily retrieving tokens that can be used in tests utilizing authentication subsystem.
func GetTestToken() (*oauth2.Token, error) {
	clientSecret := os.Getenv(envClientSecret)
	username := os.Getenv(envUsername)
	password := os.Getenv(envPassword)
	if clientSecret == "" {
		return nil, fmt.Errorf(msgMissingEnv, envClientSecret)
	}
	if username == "" {
		return nil, fmt.Errorf(msgMissingEnv, envUsername)
	}
	if password == "" {
		return nil, fmt.Errorf(msgMissingEnv, envPassword)
	}

	ctx := context.Background()
	conf := &oauth2.Config{
		// ClientID:     config.GetOauthClientID(),
		ClientID:     TestClientID,
		ClientSecret: clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: config.GetOauthTokenURL()},
	}
	return conf.PasswordCredentialsToken(ctx, username, password)
}

func GetTestTokenHeader() (string, error) {
	t, err := GetTestToken()
	if err != nil {
		return "", err
	}
	return TokenPrefix + t.AccessToken, nil
}