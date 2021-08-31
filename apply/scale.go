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

package apply

import (
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Scales interface {
	Scale(cluster *v1.Cluster, scalingArgs *common.RunArgs) error
}

func NewScalingApplierFromArgs(clusterfile string, scalingArgs *common.RunArgs, isExpand bool) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile parsing failed, please check:", err)
		return nil
	}
	if scalingArgs.Nodes == "" && scalingArgs.Masters == "" {
		logger.Error("The node or master parameter was not committed")
		return nil
	}
	var err error
	if isExpand {
		e := Expand{}
		err = e.Scale(cluster, scalingArgs)
	} else {
		s := Shrink{}
		err = s.Scale(cluster, scalingArgs)
	}
	if err != nil {
		logger.Error(err)
		return nil
	}
	if err := utils.MarshalYamlToFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile save failed, please check:", err)
		return nil
	}
	applier, err := NewApplier(cluster)
	if err != nil {
		logger.Error("failed to init applier, err: %s", err)
		return nil
	}
	return applier
}
