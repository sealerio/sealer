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

package utils

import (
	"fmt"
	"net"
	"strconv"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"

	netutils "github.com/sealerio/sealer/utils/net"
	strUtils "github.com/sealerio/sealer/utils/strings"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func ConstructClusterForRun(imageName string, runArgs *types.Args) (*v2.Cluster, error) {
	resultHosts, err := TransferIPStrToHosts(runArgs.Masters, runArgs.Nodes)
	if err != nil {
		return nil, err
	}

	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			SSH: v1.SSH{
				User:     runArgs.User,
				Passwd:   runArgs.Password,
				PkPasswd: runArgs.PkPassword,
				Pk:       runArgs.Pk,
				Port:     strconv.Itoa(int(runArgs.Port)),
			},
			Image:   imageName,
			Hosts:   resultHosts,
			Env:     runArgs.CustomEnv,
			CMDArgs: runArgs.CMDArgs,
		},
	}
	cluster.APIVersion = common.APIVersion
	cluster.Kind = common.Kind
	cluster.Name = "my-cluster"
	return &cluster, nil
}

func ConstructClusterForJoin(cluster *v2.Cluster, scaleArgs *types.Args, joinMasters, joinWorkers []net.IP) error {
	// merge custom Env to the existed cluster
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	//todo Add password encryption mode in the future

	//add joined masters
	if len(joinMasters) != 0 {
		masterIPs := cluster.GetMasterIPList()
		for _, ip := range joinMasters {
			// if ip already taken by master will return join duplicated ip error
			if !netutils.NotInIPList(ip, masterIPs) {
				return fmt.Errorf("failed to scale master for duplicated ip: %s", ip)
			}
		}

		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = append(cluster.Spec.Hosts[i].IPS, joinMasters...)
			}
		}

		var hosts []v2.Host
		for _, h := range cluster.Spec.Hosts {
			if len(h.IPS) == 0 {
				continue
			}
			hosts = append(hosts, h)
		}

		cluster.Spec.Hosts = hosts
	}

	//add joined nodes
	if len(joinWorkers) != 0 {
		nodeIPs := cluster.GetNodeIPList()
		for _, ip := range joinWorkers {
			// if ip already taken by node will return join duplicated ip error
			if !netutils.NotInIPList(ip, nodeIPs) {
				return fmt.Errorf("failed to scale node for duplicated ip: %s", ip)
			}
		}

		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.NODE, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = append(cluster.Spec.Hosts[i].IPS, joinWorkers...)
			}
		}

		var hosts []v2.Host
		for _, h := range cluster.Spec.Hosts {
			if len(h.IPS) == 0 {
				continue
			}
			hosts = append(hosts, h)
		}

		cluster.Spec.Hosts = hosts
	}
	return nil
}

func ConstructClusterForScaleDown(cluster *v2.Cluster, mastersToDelete, workersToDelete []net.IP) error {
	if len(mastersToDelete) != 0 {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, mastersToDelete)
			}
		}
	}

	if len(workersToDelete) != 0 {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.NODE, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, workersToDelete)
			}
		}
	}

	// if hosts have no ip address exist,then delete this host.
	var hosts []v2.Host
	for _, host := range cluster.Spec.Hosts {
		if len(host.IPS) == 0 {
			continue
		}
		hosts = append(hosts, host)
	}
	cluster.Spec.Hosts = hosts

	return nil
}

func returnFilteredIPList(clusterIPList []net.IP, toBeDeletedIPList []net.IP) (res []net.IP) {
	for _, ip := range clusterIPList {
		if netutils.NotInIPList(ip, toBeDeletedIPList) {
			res = append(res, ip)
		}
	}
	return
}
