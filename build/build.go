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

package build

import (
	"fmt"

	"github.com/alibaba/sealer/build/cloud"
	"github.com/alibaba/sealer/build/lite"
	"github.com/alibaba/sealer/build/local"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/store"
)

var ProviderMap = map[string]string{
	common.LocalBuild:     common.BAREMETAL,
	common.AliCloudBuild:  common.AliCloud,
	common.ContainerBuild: common.CONTAINER,
}

func NewLocalBuilder(config *Config) (Interface, error) {
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	service, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	fs, err := store.NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init store backend, err: %s", err)
	}

	prober := image.NewImageProber(service, config.NoCache)

	return &local.Builder{
		BuildType:    config.BuildType,
		NoCache:      config.NoCache,
		LayerStore:   layerStore,
		ImageStore:   imageStore,
		ImageService: service,
		Prober:       prober,
		FS:           fs,
	}, nil
}

func NewCloudBuilder(config *Config) (Interface, error) {
	localBuilder, err := NewLocalBuilder(config)
	if err != nil {
		return nil, err
	}

	provider := common.AliCloud
	if config.BuildType != "" {
		provider = ProviderMap[config.BuildType]
	}

	return &cloud.Builder{
		Local:              localBuilder.(*local.Builder),
		Provider:           provider,
		TmpClusterFilePath: common.TmpClusterfile,
	}, nil
}

func NewLiteBuilder(config *Config) (Interface, error) {
	localBuilder, err := NewLocalBuilder(config)
	if err != nil {
		return nil, err
	}

	return &lite.Builder{
		Local: localBuilder.(*local.Builder),
	}, nil
}
