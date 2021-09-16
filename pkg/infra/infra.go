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

package infra

import (
	"fmt"
	aliyun2 "github.com/alibaba/sealer/pkg/infra/aliyun"
	container2 "github.com/alibaba/sealer/pkg/infra/container"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Interface interface {
	// Apply apply iaas resources and save metadata info like vpc instance id to cluster status
	// https://github.com/fanux/sealgate/tree/master/cloud
	Apply() error
}

func NewDefaultProvider(cluster *v1.Cluster) (Interface, error) {
	switch cluster.Spec.Provider {
	case aliyun2.AliCloud:
		return NewAliProvider(cluster)
	case container2.CONTAINER:
		return NewContainerProvider(cluster)
	default:
		return nil, fmt.Errorf("the provider is invalid, please set the provider correctly")
	}
}

func NewAliProvider(cluster *v1.Cluster) (Interface, error) {
	config := new(aliyun2.Config)
	err := aliyun2.LoadConfig(config)
	if err != nil {
		return nil, err
	}
	aliProvider := new(aliyun2.AliProvider)
	aliProvider.Config = *config
	aliProvider.Cluster = cluster
	err = aliProvider.NewClient()
	if err != nil {
		return nil, err
	}
	return aliProvider, nil
}

func NewContainerProvider(cluster *v1.Cluster) (Interface, error) {
	if container2.IsDockerAvailable() {
		return nil, fmt.Errorf("please install docker on your system")
	}

	cli, err := container2.NewClientWithCluster(cluster)
	if err != nil {
		return nil, fmt.Errorf("new container client failed")
	}

	return cli, nil
}
