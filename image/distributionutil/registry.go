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

	"github.com/alibaba/sealer/logger"

	"github.com/docker/docker/api/types"

	"github.com/docker/distribution"

	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/utils"
)

func NewV2Repository(named reference.Named, actions ...string) (distribution.Repository, error) {
	authConfig := types.AuthConfig{ServerAddress: named.Domain()}
	repo, err := getV2Repository(authConfig, named, actions...)
	if err == nil {
		return repo, nil
	}

	authConfig, authErr := utils.GetDockerAuthInfoFromDocker(named.Domain())
	if authErr != nil {
		logger.Debug("failed to get auth info, err: %s", authErr)
		return nil, err
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
