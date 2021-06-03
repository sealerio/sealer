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
	"github.com/alibaba/sealer/infra/aliyun"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Interface interface {
	// Apply apply iaas resources and save metadata info like vpc instance id to cluster status
	// https://github.com/fanux/sealgate/tree/master/cloud
	Apply() error
}

func NewDefaultProvider(cluster *v1.Cluster) Interface {
	switch cluster.Spec.Provider {
	case aliyun.AliCloud:
		config := new(aliyun.Config)
		err := aliyun.LoadConfig(config)
		if err != nil {
			logger.Error(err)
			return nil
		}
		aliProvider := new(aliyun.AliProvider)
		aliProvider.Config = *config
		aliProvider.Cluster = cluster
		err = aliProvider.NewClient()
		if err != nil {
			logger.Error(err)
		}
		return aliProvider
	default:
		return nil
	}
}
