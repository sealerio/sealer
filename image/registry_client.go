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

package image

import (
	"context"

	"github.com/alibaba/sealer/logger"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"

	"github.com/alibaba/sealer/registry"
	"github.com/docker/docker/api/types"
)

func initRegistry(hostname string) (*registry.Registry, error) {
	var (
		authInfo types.AuthConfig
		err      error
		reg      *registry.Registry
	)

	authInfo, err = pkgutils.GetDockerAuthInfoFromDocker(hostname)
	if err != nil {
		logger.Warn("failed to get auth info for %s, err: %s", hostname, err)
	}

	reg, err = fetchRegistryClient(authInfo)
	if err != nil {
		err = errors.Wrap(err, "failed to fetch registry client")
		return nil, err
	}
	return reg, err
}

//fetch https and http registry client
func fetchRegistryClient(auth types.AuthConfig) (*registry.Registry, error) {
	reg, err := registry.New(context.Background(), auth, registry.Opt{Insecure: true})
	if err == nil {
		return reg, nil
	}
	reg, err = registry.New(context.Background(), auth, registry.Opt{Insecure: true, NonSSL: true})
	if err == nil {
		return reg, nil
	}
	return nil, err
}
