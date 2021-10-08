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

package docker

import (
	"encoding/base64"
	"encoding/json"

	normalreference "github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dockerregistry "github.com/docker/docker/registry"
)

func GetCanonicalImageName(rawImageName string) reference.Named {
	var named reference.Named
	named, err := reference.ParseNormalizedNamed(rawImageName)
	if err != nil {
		logger.Warn("parse canonical image name failed: %v", err)
		return named
	}
	return named
}

func GetCanonicalImagePullOptions(canonicalImageName string) types.ImagePullOptions {
	var (
		err         error
		authConfig  types.AuthConfig
		encodedJSON []byte
		authStr     string
		opts        types.ImagePullOptions
	)

	named, err := normalreference.ParseToNamed(canonicalImageName)

	if err != nil {
		logger.Warn("parse canonical ImageName failed: %v", err)
		return opts
	}

	//convert default docker.io to its default index server endpoint
	registryAddr := named.Domain()
	if registryAddr == dockerregistry.IndexName {
		registryAddr = dockerregistry.IndexServer
	}

	authConfig, err = utils.GetDockerAuthInfoFromDocker(registryAddr)
	if err == nil {
		encodedJSON, err = json.Marshal(authConfig)
		if err != nil {
			logger.Warn("authConfig encodedJSON failed: %v", err)
		} else {
			authStr = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}

	return types.ImagePullOptions{RegistryAuth: authStr}

}
