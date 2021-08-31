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
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Expand struct {
}

func (p *Expand) Scale(cluster *v1.Cluster, scalingArgs *common.RunArgs) error {
	switch cluster.Spec.Provider {
	case common.BAREMETAL:
		if err := PreProcessIPList(scalingArgs); err != nil {
			return err
		}
		if (!IsIPList(scalingArgs.Nodes) && scalingArgs.Nodes != "") || (!IsIPList(scalingArgs.Masters) && scalingArgs.Masters != "") {
			return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
		}
		if scalingArgs.Nodes != "" && IsIPList(scalingArgs.Nodes) && scalingArgs.Masters != "" && IsIPList(scalingArgs.Masters) {
			margeNodes := append(cluster.Spec.Nodes.IPList, strings.Split(scalingArgs.Nodes, ",")...)
			margeMasters := append(cluster.Spec.Masters.IPList, strings.Split(scalingArgs.Masters, ",")...)
			cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
			cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
			return nil
		}
		if scalingArgs.Nodes == "" && scalingArgs.Masters != "" && IsIPList(scalingArgs.Masters) {
			margeMasters := append(cluster.Spec.Masters.IPList, strings.Split(scalingArgs.Masters, ",")...)
			cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
			return nil
		}
		if scalingArgs.Nodes != "" && scalingArgs.Masters == "" && IsIPList(scalingArgs.Nodes) {
			margeNodes := append(cluster.Spec.Nodes.IPList, strings.Split(scalingArgs.Nodes, ",")...)
			cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
			return nil
		}
		return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
	case common.AliCloud:
		if (!IsNumber(scalingArgs.Nodes) && scalingArgs.Nodes != "") || (!IsNumber(scalingArgs.Masters) && scalingArgs.Masters != "") {
			return fmt.Errorf(" Parameter error: The number of join masters or nodes that must be submitted to use cloud service！")
		}
		if scalingArgs.Nodes != "" && IsNumber(scalingArgs.Nodes) && scalingArgs.Masters != "" && IsNumber(scalingArgs.Masters) {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) + StrToInt(scalingArgs.Masters))
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) + StrToInt(scalingArgs.Nodes))
			return nil
		}
		if scalingArgs.Nodes == "" && scalingArgs.Masters != "" && IsNumber(scalingArgs.Masters) {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) + StrToInt(scalingArgs.Masters))
			return nil
		}
		if scalingArgs.Nodes != "" && scalingArgs.Masters == "" && IsNumber(scalingArgs.Nodes) {
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) + StrToInt(scalingArgs.Nodes))
			return nil
		}
		return fmt.Errorf(" Parameter error: The number of join masters or nodes that must be submitted to use cloud service！")
	default:
		return fmt.Errorf(" clusterfile provider type is not found ！")
	}
}

func StrToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		logger.Error("String to digit conversion failed:", err)
		return 0
	}
	return num
}

func removeIPListDuplicatesAndEmpty(ipList []string) []string {
	count := len(ipList)
	var newList []string
	for i := 0; i < count; i++ {
		if (i > 0 && ipList[i-1] == ipList[i]) || len(ipList[i]) == 0 {
			continue
		}
		newList = append(newList, ipList[i])
	}
	return newList
}
