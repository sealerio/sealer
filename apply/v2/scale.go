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

	"github.com/alibaba/sealer/apply/v2/applytype"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

// NewScaleApplierFromArgs will filter ip list from command parameters.
func NewScaleApplierFromArgs(clusterfile string, scaleArgs *common.RunArgs, flag string) (applytype.Interface, error) {
	cluster := &v2.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		return nil, err
	}
	if scaleArgs.Nodes == "" && scaleArgs.Masters == "" {
		return nil, fmt.Errorf("the node or master parameter was not committed")
	}

	var err error
	switch flag {
	case common.JoinSubCmd:
		err = Join(cluster, scaleArgs)
	case common.DeleteSubCmd:
		err = Delete(cluster, scaleArgs)
	}
	if err != nil {
		return nil, err
	}

	/*	if err := utils.MarshalYamlToFile(clusterfile, cluster); err != nil {
		return nil, err
	}*/
	applier, err := NewApplier(cluster)
	if err != nil {
		return nil, err
	}
	return applier, nil
}

func Join(cluster *v2.Cluster, scalingArgs *common.RunArgs) error {
	/*	switch cluster.Spec.Provider {
		case common.BAREMETAL:
			return joinBaremetalNodes(cluster, scalingArgs)
		case common.AliCloud:
			return joinInfraNodes(cluster, scalingArgs)
		case common.CONTAINER:
			return joinInfraNodes(cluster, scalingArgs)
		default:
			return fmt.Errorf(" clusterfile provider type is not found ！")
		}*/
	return joinBaremetalNodes(cluster, scalingArgs)
}

func joinBaremetalNodes(cluster *v2.Cluster, scaleArgs *common.RunArgs) error {
	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}
	if (!IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
	}
	// join nodes cannot be in the current cluster
	if len(utils.ReduceIPList(removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Masters, ",")), cluster.GetMasterIPList())) != 0 ||
		len(utils.ReduceIPList(removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Nodes, ",")), cluster.GetNodeIPList())) != 0 {
		return fmt.Errorf("join nodes already in the current cluster")
	}

	if scaleArgs.Masters != "" && IsIPList(scaleArgs.Masters) {
		for i := 0; i < len(cluster.Spec.Hosts); i++ {
			role := cluster.Spec.Hosts[i].Roles
			if utils.InList(common.MASTER, role) {
				cluster.Spec.Hosts[i].IPS = append(cluster.Spec.Hosts[i].IPS, removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Masters, ","))...)
				break
			}
			if i == len(cluster.Spec.Hosts)-1 {
				return fmt.Errorf("not found `master` role from file")
			}
		}
	}
	//add join node
	if scaleArgs.Nodes != "" && IsIPList(scaleArgs.Nodes) {
		for i := 0; i < len(cluster.Spec.Hosts); i++ {
			role := cluster.Spec.Hosts[i].Roles
			if utils.InList(common.NODE, role) {
				cluster.Spec.Hosts[i].IPS = append(cluster.Spec.Hosts[i].IPS, removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Nodes, ","))...)
				break
			}
			if i == len(cluster.Spec.Hosts)-1 {
				hosts := v2.Host{IPS: removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Nodes, ",")), Roles: []string{common.NODE}}
				cluster.Spec.Hosts = append(cluster.Spec.Hosts, hosts)
			}
		}
	}
	return nil
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

func Delete(cluster *v2.Cluster, scaleArgs *common.RunArgs) error {
	return deleteBaremetalNodes(cluster, scaleArgs)
}

func deleteBaremetalNodes(cluster *v2.Cluster, scaleArgs *common.RunArgs) error {
	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}
	if (!IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The current mode should submit iplist！")
	}
	//delete node must be in the current cluster
	if len(utils.RemoveIPList(removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Masters, ",")), cluster.GetMasterIPList())) != 0 ||
		len(utils.RemoveIPList(removeIPListDuplicatesAndEmpty(strings.Split(scaleArgs.Nodes, ",")), cluster.GetNodeIPList())) != 0 {
		return fmt.Errorf("delete nodes are not in the current cluster")
	}

	if scaleArgs.Masters != "" && IsIPList(scaleArgs.Masters) {
		for i := range cluster.Spec.Hosts {
			if utils.InList(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Masters, ","))
			}
		}
	}
	if scaleArgs.Nodes != "" && IsIPList(scaleArgs.Nodes) {
		for i := range cluster.Spec.Hosts {
			if utils.InList(common.NODE, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Nodes, ","))
			}
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
