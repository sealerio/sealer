package distributionutil

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/dockerversion"
	dockerRegistry "github.com/docker/docker/registry"
)

func Login(ctx context.Context, authConfig *types.AuthConfig) error {
	domain := authConfig.ServerAddress
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	endpointURL, err := url.Parse(domain)
	if err != nil {
		return err
	}

	modifiers := dockerRegistry.Headers(dockerversion.DockerUserAgent(ctx), nil)
	authTransport := transport.NewTransport(dockerRegistry.NewTransport(nil), modifiers...)

	credentialAuthConfig := *authConfig
	creds := loginCredentialStore{
		authConfig: &credentialAuthConfig,
	}
	loginClient, err := authHTTPClient(endpointURL, authTransport, modifiers, creds, nil)
	if err != nil {
		return err
	}

	endpointStr := strings.TrimRight(endpointURL.String(), "/") + "/v2/"
	req, err := http.NewRequest("GET", endpointStr, nil)
	if err != nil {
		return err
	}

	resp, err := loginClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// TODO(dmcgowan): Attempt to further interpret result, status code and error code string
	err = errors.Errorf("login attempt to %s failed with status: %d %s", endpointStr, resp.StatusCode, http.StatusText(resp.StatusCode))
	return err
}

func authHTTPClient(endpoint *url.URL, authTransport http.RoundTripper, modifiers []transport.RequestModifier, creds auth.CredentialStore, scopes []auth.Scope) (*http.Client, error) {
	challengeManager, _, err := dockerRegistry.PingV2Registry(endpoint, authTransport)
	if err != nil {
		return nil, err
	}

	tokenHandlerOptions := auth.TokenHandlerOptions{
		Transport:     authTransport,
		Credentials:   creds,
		OfflineAccess: true,
		ClientID:      dockerRegistry.AuthClientID,
		Scopes:        scopes,
	}
	tokenHandler := auth.NewTokenHandlerWithOptions(tokenHandlerOptions)
	basicHandler := auth.NewBasicHandler(creds)
	modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler))
	tr := transport.NewTransport(authTransport, modifiers...)

	return &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}, nil
}

type loginCredentialStore struct {
	authConfig *types.AuthConfig
}

func (lcs loginCredentialStore) Basic(*url.URL) (string, string) {
	return lcs.authConfig.Username, lcs.authConfig.Password
}

func (lcs loginCredentialStore) RefreshToken(*url.URL, string) string {
	return lcs.authConfig.IdentityToken
}

func (lcs loginCredentialStore) SetRefreshToken(u *url.URL, service, token string) {
	lcs.authConfig.IdentityToken = token
}
