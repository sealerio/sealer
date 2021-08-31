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
)

type Shrink struct {
}

func (p *Shrink) Scale(cluster *v1.Cluster, scalingArgs *common.RunArgs) error {
	switch cluster.Spec.Provider {
	case common.BAREMETAL:
		if err := PreProcessIPList(scalingArgs); err != nil {
			return err
		}
		if (!IsIPList(scalingArgs.Nodes) && scalingArgs.Nodes != "") || (!IsIPList(scalingArgs.Masters) && scalingArgs.Masters != "") {
			return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
		}
		if scalingArgs.Nodes != "" && IsIPList(scalingArgs.Nodes) && scalingArgs.Masters != "" && IsIPList(scalingArgs.Masters) {
			margeMasters := returnFilteredIPList(cluster.Spec.Masters.IPList, strings.Split(scalingArgs.Masters, ","))
			margeNodes := returnFilteredIPList(cluster.Spec.Nodes.IPList, strings.Split(scalingArgs.Nodes, ","))
			cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
			cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
			return nil
		}
		if scalingArgs.Nodes == "" && scalingArgs.Masters != "" && IsIPList(scalingArgs.Masters) {
			margeMasters := returnFilteredIPList(cluster.Spec.Masters.IPList, strings.Split(scalingArgs.Masters, ","))
			cluster.Spec.Masters.IPList = removeIPListDuplicatesAndEmpty(margeMasters)
			return nil
		}
		if scalingArgs.Nodes != "" && scalingArgs.Masters == "" && IsIPList(scalingArgs.Nodes) {
			margeNodes := returnFilteredIPList(cluster.Spec.Nodes.IPList, strings.Split(scalingArgs.Nodes, ","))
			cluster.Spec.Nodes.IPList = removeIPListDuplicatesAndEmpty(margeNodes)
			return nil
		}
		return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
	case common.AliCloud:
		if (!IsNumber(scalingArgs.Nodes) && scalingArgs.Nodes != "") || (!IsNumber(scalingArgs.Masters) && scalingArgs.Masters != "") {
			return fmt.Errorf(" Parameter error: The number of join masters or nodes that must be submitted to use cloud service！")
		}
		if scalingArgs.Nodes != "" && IsNumber(scalingArgs.Nodes) && scalingArgs.Masters != "" && IsNumber(scalingArgs.Masters) {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) - StrToInt(scalingArgs.Masters))
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) - StrToInt(scalingArgs.Nodes))
			if StrToInt(cluster.Spec.Masters.Count) <= 0 || StrToInt(cluster.Spec.Nodes.Count) <= 0 {
				return fmt.Errorf("parameter error: the number of clean masters or nodes that must be less than definition in Clusterfile")
			}
			return nil
		}
		if scalingArgs.Nodes == "" && scalingArgs.Masters != "" && IsNumber(scalingArgs.Masters) {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) - StrToInt(scalingArgs.Masters))
			if StrToInt(cluster.Spec.Masters.Count) <= 0 {
				return fmt.Errorf("parameter error: the number of clean masters or nodes that must be less than definition in Clusterfile")
			}
			return nil
		}
		if scalingArgs.Nodes != "" && scalingArgs.Masters == "" && IsNumber(scalingArgs.Nodes) {
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) - StrToInt(scalingArgs.Nodes))
			if StrToInt(cluster.Spec.Nodes.Count) <= 0 {
				return fmt.Errorf("parameter error: the number of clean masters or nodes that must be less than definition in Clusterfile")
			}
			return nil
		}
		return fmt.Errorf(" Parameter error: The number of join masters or nodes that must be submitted to use cloud service！")
	default:
		return fmt.Errorf(" clusterfile provider type is not found ！")
	}
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
