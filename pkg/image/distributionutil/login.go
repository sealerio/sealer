// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package distributionutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/dockerversion"
	dockerRegistry "github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	base := dockerRegistry.NewTransport(nil)
	base.TLSClientConfig.InsecureSkipVerify = os.Getenv("SKIP_TLS_VERIFY") == "true"
	if err := dockerRegistry.ReadCertsDirectory(base.TLSClientConfig, filepath.Join(dockerRegistry.CertsDir(), endpointURL.Host)); err != nil {
		return err
	}
	authTransport := transport.NewTransport(base, modifiers...)

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
		if strings.Contains(err.Error(), "x509") {
			return fmt.Errorf("%v, if you want to skip TLS verification, set the environment variable 'SKIP_TLS_VERIFY=true' ", err)
		}
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Warnf("failed to close http reader")
		}
	}(resp.Body)

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
