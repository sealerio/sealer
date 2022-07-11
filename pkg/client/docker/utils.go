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

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dockerregistry "github.com/docker/docker/registry"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/client/docker/auth"
	normalreference "github.com/sealerio/sealer/pkg/image/reference"
)

func GetCanonicalImageName(rawImageName string) (reference.Named, error) {
	var named reference.Named
	named, err := reference.ParseNormalizedNamed(rawImageName)
	if err != nil {
		return nil, err
	}
	return named, nil
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
		logrus.Warnf("failed to parse canonical ImageName: %v", err)
		return opts
	}

	//convert default docker.io to its default index server endpoint
	registryAddr := named.Domain()
	if registryAddr == dockerregistry.IndexName {
		registryAddr = dockerregistry.IndexServer
	}
	svc, err := auth.NewDockerAuthService()
	if err != nil {
		return opts
	}

	authConfig, err = svc.GetAuthByDomain(registryAddr)
	if err == nil {
		encodedJSON, err = json.Marshal(authConfig)
		if err != nil {
			logrus.Warnf("failed to authConfig encodedJSON: %v", err)
		} else {
			authStr = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}
	return types.ImagePullOptions{RegistryAuth: authStr}
}
