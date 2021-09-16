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
	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/pkg/logger"

	"github.com/alibaba/sealer/utils"

	"github.com/docker/distribution"
)

func NewV2Repository(named reference.Named, actions ...string) (distribution.Repository, error) {
	authConfig, err := utils.GetDockerAuthInfoFromDocker(named.Domain())
	if err != nil {
		logger.Warn("failed to get auth info, err: %s", err)
	}

	repo, err := NewRepository(context.Background(), authConfig, named.Repo(), registryConfig{Insecure: true, Domain: named.Domain()}, actions...)
	if err == nil {
		return repo, nil
	}

	return NewRepository(context.Background(), authConfig, named.Repo(), registryConfig{Insecure: true, NonSSL: true, Domain: named.Domain()}, actions...)
}
