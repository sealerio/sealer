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

// NewCleanApplierFromArgs will filter ip list from command parameters.
func NewCleanApplierFromArgs(clusterfile string, cleanArgs *common.RunArgs, cleanOpts *common.RunOpts) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile parsing failed, please check:", err)
		return nil
	}
	if cleanArgs.Nodes == "" && cleanArgs.Masters == "" && !cleanOpts.All {
		logger.Error("The node or master parameter was not committed")
		return nil
	}
	if cluster.Spec.Provider == "BAREMETAL" {
		if cleanOpts.All {
			cluster.Spec.Masters.IPList = cluster.Spec.Masters.IPList[0:0]
			cluster.Spec.Nodes.IPList = cluster.Spec.Nodes.IPList[0:0]
		} else {
			if err := PreProcessIPList(cleanArgs); err != nil {
				logger.Error("please check you ips format:", err)
				return nil
			}
			if IsIPList(cleanArgs.Nodes) || IsIPList(cleanArgs.Masters) {
				margeMasters := returnFilteredIPList(cluster.Spec.Masters.IPList, strings.Split(cleanArgs.Masters, ","))
				margeNodes := returnFilteredIPList(cluster.Spec.Nodes.IPList, strings.Split(cleanArgs.Nodes, ","))
				cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
				cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
			} else {
				logger.Error("Parameter error:", "The current mode should submit iplist！")
				return nil
			}
		}
	} else if IsNumber(cleanArgs.Nodes) || IsNumber(cleanArgs.Masters) {
		if cleanOpts.All {
			cluster.Spec.Masters.Count = "0"
			cluster.Spec.Nodes.Count = "0"
		} else {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) - StrToInt(cleanArgs.Masters))
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) - StrToInt(cleanArgs.Nodes))
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
	cleanIPMap := make(map[string]bool)
	for _, cleanIP := range toBeDeletedIPList {
		cleanIPMap[cleanIP] = true
	}

	for i := 0; i < len(clusterIPList); i++ {
		if _, ok := cleanIPMap[clusterIPList[i]]; ok {
			clusterIPList = append(clusterIPList[:i], clusterIPList[i+1:]...)
			i = i - 1
		}
	}
	return clusterIPList
}
