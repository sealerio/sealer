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
	"strconv"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// NewScalingApplierFromArgs will filter ip list from command parameters.
func NewScalingApplierFromArgs(clusterfile string, scalingArgs *common.RunArgs) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile parsing failed, please check:", err)
		return nil
	}
	if scalingArgs.Nodes == "" && scalingArgs.Masters == "" {
		logger.Error("The node or master parameter was not committed")
		return nil
	}
	if cluster.Spec.Provider == "BAREMETAL" {
		if err := PreProcessIPList(scalingArgs); err != nil {
			logger.Error("please check you ips format:", err)
			return nil
		}
		if IsIPList(scalingArgs.Nodes) || IsIPList(scalingArgs.Masters) {
			margeMasters := returnFilteredIPList(cluster.Spec.Masters.IPList, strings.Split(scalingArgs.Masters, ","))
			margeNodes := returnFilteredIPList(cluster.Spec.Nodes.IPList, strings.Split(scalingArgs.Nodes, ","))
			cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
			cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
		} else {
			logger.Error("Parameter error:", "The current mode should submit iplist！")
			return nil
		}
	} else if IsNumber(scalingArgs.Nodes) || IsNumber(scalingArgs.Masters) {
		cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) - StrToInt(scalingArgs.Masters))
		cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) - StrToInt(scalingArgs.Nodes))
		if StrToInt(cluster.Spec.Masters.Count) <= 0 || StrToInt(cluster.Spec.Nodes.Count) <= 0 {
			logger.Error("Parameter error:", "The number of clean masters or nodes that must be less than definition in Clusterfile.")
			return nil
		}
	} else {
		logger.Error("Parameter error:", "The number of clean masters or nodes that must be submitted to use cloud service！")
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

func returnFilteredIPList(clusterIPList []string, toBeDeletedIPList []string) []string {
	f := make(map[string]byte)
	s := make(map[string]byte)

	var set []string

	for _, v := range clusterIPList {
		f[v] = 0
		s[v] = 0
	}
	for _, v := range toBeDeletedIPList {
		l := len(s)
		s[v] = 1
		if l == len(s) {
			set = append(set, v)
		}
	}
	for _, v := range set {
		delete(s, v)
	}
	var result []string
	for v := range s {
		_, exist := f[v]
		if exist {
			result = append(result, v)
		}
	}

	return result
}
