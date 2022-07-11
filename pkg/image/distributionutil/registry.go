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

	"github.com/sealerio/sealer/pkg/client/docker/auth"

	"github.com/distribution/distribution/v3"
	"github.com/docker/docker/api/types"

	"github.com/sealerio/sealer/pkg/image/reference"
)

func NewV2Repository(named reference.Named, actions ...string) (distribution.Repository, error) {
	var (
		domain      = named.Domain()
		defaultAuth = types.AuthConfig{ServerAddress: domain}
	)

	svc, err := auth.NewDockerAuthService()
	if err != nil {
		return nil, fmt.Errorf("failed to read default auth file: %v", err)
	}

	authConfig, err := svc.GetAuthByDomain(domain)
	if err != nil && authConfig != defaultAuth {
		return nil, fmt.Errorf("failed to get auth info for domain(%s): %v", domain, err)
	}

	return getV2Repository(authConfig, named, actions...)
}

func getV2Repository(authConfig types.AuthConfig, named reference.Named, actions ...string) (distribution.Repository, error) {
	repo, err := NewRepository(context.Background(), authConfig, named.Repo(), registryConfig{Insecure: true, Domain: named.Domain()}, actions...)
	if err == nil {
		return repo, nil
	}
	return NewRepository(context.Background(), authConfig, named.Repo(), registryConfig{Insecure: true, NonSSL: true, Domain: named.Domain()}, actions...)
}
