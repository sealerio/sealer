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
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/distribution/distribution/v3"
	dockerRegistryClient "github.com/distribution/distribution/v3/registry/client"
	"github.com/docker/distribution/reference"
	dockerAuth "github.com/docker/distribution/registry/client/auth"
	dockerTransport "github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/dockerversion"
	dockerRegistry "github.com/docker/docker/registry"
	"github.com/docker/go-connections/tlsconfig"
)

func NewRepository(ctx context.Context, authConfig types.AuthConfig, repoName string, config registryConfig, actions ...string) (distribution.Repository, error) {
	tlsConfig := tlsconfig.ServerDefault()
	tlsConfig.InsecureSkipVerify = config.Insecure

	rurlStr := strings.TrimSuffix(config.Domain, "/")
	if !strings.HasPrefix(rurlStr, "https://") && !strings.HasPrefix(rurlStr, "http://") {
		if !config.NonSSL {
			rurlStr = "https://" + rurlStr
		} else {
			rurlStr = "http://" + rurlStr
		}
	}

	rurl, err := url.Parse(rurlStr)
	if err != nil {
		return nil, err
	}

	direct := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	// TODO(dmcgowan): Call close idle connections when complete, use keep alive
	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         direct.DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
		// TODO(dmcgowan): Call close idle connections when complete and use keep alive
		DisableKeepAlives: true,
	}
	if err := dockerRegistry.ReadCertsDirectory(base.TLSClientConfig, filepath.Join(dockerRegistry.CertsDir(), rurl.Host)); err != nil {
		return nil, err
	}
	modifiers := dockerRegistry.Headers(dockerversion.DockerUserAgent(ctx), nil)
	authTransport := dockerTransport.NewTransport(base, modifiers...)

	challengeManager, _, err := dockerRegistry.PingV2Registry(rurl, authTransport)
	if err != nil {
		return nil, err
	}
	// typically, this filed would be empty
	if authConfig.RegistryToken != "" {
		passThruTokenHandler := &existingTokenHandler{token: authConfig.RegistryToken}
		modifiers = append(modifiers, dockerAuth.NewAuthorizer(challengeManager, passThruTokenHandler))
	} else {
		scope := dockerAuth.RepositoryScope{
			Repository: repoName,
			Actions:    actions,
			Class:      "image",
		}

		creds := dockerRegistry.NewStaticCredentialStore(&authConfig)
		tokenHandlerOptions := dockerAuth.TokenHandlerOptions{
			Transport:   authTransport,
			Credentials: creds,
			Scopes:      []dockerAuth.Scope{scope},
			ClientID:    dockerRegistry.AuthClientID,
		}
		tokenHandler := dockerAuth.NewTokenHandlerWithOptions(tokenHandlerOptions)
		basicHandler := dockerAuth.NewBasicHandler(creds)
		modifiers = append(modifiers, dockerAuth.NewAuthorizer(challengeManager, tokenHandler, basicHandler))
	}

	tr := dockerTransport.NewTransport(base, modifiers...)
	repoNameRef, err := reference.WithName(repoName)
	if err != nil {
		return nil, err
	}

	return dockerRegistryClient.NewRepository(repoNameRef, rurl.String(), tr)
}

type existingTokenHandler struct {
	token string
}

func (th *existingTokenHandler) Scheme() string {
	return "bearer"
}

func (th *existingTokenHandler) AuthorizeRequest(req *http.Request, params map[string]string) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", th.token))
	return nil
}
