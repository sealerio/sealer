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
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func Delete(cluster *v1.Cluster, scaleArgs *common.RunArgs) error {
	switch cluster.Spec.Provider {
	case common.BAREMETAL:
		return deleteBaremetalNodes(cluster, scaleArgs)
	case common.AliCloud:
		return deleteInfraNodes(cluster, scaleArgs)
	case common.CONTAINER:
		return deleteInfraNodes(cluster, scaleArgs)
	default:
		return fmt.Errorf(" clusterfile provider type is not found ！")
	}
}

func deleteBaremetalNodes(cluster *v1.Cluster, scaleArgs *common.RunArgs) error {
	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}
	if (!IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
	}
	if scaleArgs.Masters != "" && IsIPList(scaleArgs.Masters) {
		margeMasters := returnFilteredIPList(cluster.Spec.Masters.IPList, strings.Split(scaleArgs.Masters, ","))
		cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
	}
	if scaleArgs.Nodes != "" && IsIPList(scaleArgs.Nodes) {
		margeNodes := returnFilteredIPList(cluster.Spec.Nodes.IPList, strings.Split(scaleArgs.Nodes, ","))
		cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
	}
	return nil
}

func deleteInfraNodes(cluster *v1.Cluster, scaleArgs *common.RunArgs) error {
	if (!IsNumber(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!IsNumber(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The number of join masters or nodes that must be submitted to use cloud service！")
	}
	if scaleArgs.Masters != "" && IsNumber(scaleArgs.Masters) {
		cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) - StrToInt(scaleArgs.Masters))
		if StrToInt(cluster.Spec.Masters.Count) <= 0 {
			return fmt.Errorf("parameter error: the number of clean masters or nodes that must be less than definition in Clusterfile")
		}
	}
	if scaleArgs.Nodes != "" && IsNumber(scaleArgs.Nodes) {
		cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) - StrToInt(scaleArgs.Nodes))
		if StrToInt(cluster.Spec.Nodes.Count) <= 0 {
			return fmt.Errorf("parameter error: the number of clean masters or nodes that must be less than definition in Clusterfile")
		}
	}
	return nil
}

func returnFilteredIPList(clusterIPList []string, toBeDeletedIPList []string) (res []string) {
	for _, ip := range clusterIPList {
		if utils.NotIn(ip, toBeDeletedIPList) {
			res = append(res, ip)
		}
	}
	return
}
