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
	"strings"

	"github.com/sealerio/sealer/utils/yaml"

	"github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/apply/applydriver"
	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
	strUtils "github.com/sealerio/sealer/utils/strings"
)

// NewScaleApplierFromArgs will filter ip list from command parameters.
func NewScaleApplierFromArgs(clusterfile string, scaleArgs *Args, flag string) (applydriver.Interface, error) {
	cluster := &v2.Cluster{}
	if err := yaml.UnmarshalFile(clusterfile, cluster); err != nil {
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
	applier, err := NewApplier(cluster, nil)
	if err != nil {
		return nil, err
	}
	return applier, nil
}

func Join(cluster *v2.Cluster, scalingArgs *Args) error {
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

func joinBaremetalNodes(cluster *v2.Cluster, scaleArgs *Args) error {
	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}
	if (!net.IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!net.IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The current mode should submit iplist!")
	}

	if net.IsIPList(scaleArgs.Masters) {
		for i := 0; i < len(cluster.Spec.Hosts); i++ {
			role := cluster.Spec.Hosts[i].Roles
			if !strUtils.NotIn(common.MASTER, role) {
				cluster.Spec.Hosts[i].IPS = removeIPListDuplicatesAndEmpty(append(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Masters, ",")...))
				break
			}
			if i == len(cluster.Spec.Hosts)-1 {
				return fmt.Errorf("not found `master` role from file")
			}
		}
	}
	//add join node
	if net.IsIPList(scaleArgs.Nodes) {
		for i := 0; i < len(cluster.Spec.Hosts); i++ {
			role := cluster.Spec.Hosts[i].Roles
			if !strUtils.NotIn(common.NODE, role) {
				cluster.Spec.Hosts[i].IPS = removeIPListDuplicatesAndEmpty(append(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Nodes, ",")...))
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

func removeIPListDuplicatesAndEmpty(ipList []string) []string {
	return strUtils.RemoveDuplicate(strUtils.NewComparator(ipList, []string{""}).GetSrcSubtraction())
}

func Delete(cluster *v2.Cluster, scaleArgs *Args) error {
	return deleteBaremetalNodes(cluster, scaleArgs)
}

func deleteBaremetalNodes(cluster *v2.Cluster, scaleArgs *Args) error {
	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}
	if (!net.IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!net.IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf(" Parameter error: The current mode should submit iplist!")
	}
	//master0 machine cannot be deleted
	if !strUtils.NotIn(cluster.GetMaster0IP(), strings.Split(scaleArgs.Masters, ",")) {
		return fmt.Errorf("master0 machine cannot be deleted")
	}
	if net.IsIPList(scaleArgs.Masters) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Masters, ","))
			}
		}
	}
	if net.IsIPList(scaleArgs.Nodes) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.NODE, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, strings.Split(scaleArgs.Nodes, ","))
			}
		}
	}
	return nil
}

func returnFilteredIPList(clusterIPList []string, toBeDeletedIPList []string) (res []string) {
	for _, ip := range clusterIPList {
		if strUtils.NotIn(ip, toBeDeletedIPList) {
			res = append(res, ip)
		}
	}
	return
}
