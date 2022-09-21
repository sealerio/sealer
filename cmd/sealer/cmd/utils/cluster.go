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

	"github.com/sealerio/sealer/utils/hash"
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
	cluster.Name = runArgs.ClusterName
	return &cluster, nil
}

func ConstructClusterForJoin(cluster *v2.Cluster, scaleArgs *types.Args, joinMasters, joinWorkers []net.IP) error {
	var err error
	// merge custom Env to the existed cluster
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	// if scaleArgs`s ssh auth credential is different from local cluster,will add it to each host.
	// if not use local cluster ssh auth credential.
	var changedSSH *v1.SSH

	passwd := cluster.Spec.SSH.Passwd
	if cluster.Spec.SSH.Encrypted {
		passwd, err = hash.AesDecrypt([]byte(cluster.Spec.SSH.Passwd))
		if err != nil {
			return err
		}
	}

	if scaleArgs.Password != "" && scaleArgs.Password != passwd {
		// Encrypt password here to avoid merge failed.
		passwd, err = hash.AesEncrypt([]byte(scaleArgs.Password))
		if err != nil {
			return err
		}
		changedSSH = &v1.SSH{
			Encrypted: true,
			User:      scaleArgs.User,
			Passwd:    passwd,
			Pk:        scaleArgs.Pk,
			PkPasswd:  scaleArgs.PkPassword,
			Port:      strconv.Itoa(int(scaleArgs.Port)),
		}
	}

	//add joined masters
	if len(joinMasters) != 0 {
		masterIPs := cluster.GetMasterIPList()
		for _, ip := range joinMasters {
			// if ip already taken by master will return join duplicated ip error
			if !netutils.NotInIPList(ip, masterIPs) {
				return fmt.Errorf("failed to scale master for duplicated ip: %s", ip)
			}
		}

		host := v2.Host{
			IPS:   joinMasters,
			Roles: []string{common.MASTER},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
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

		host := v2.Host{
			IPS:   joinWorkers,
			Roles: []string{common.NODE},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
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

func ParseScaleDownArgs(cluster *v2.Cluster, scaleArgs *types.Args, mastersToDelete []net.IP) error {
	//master0 machine cannot be deleted
	if !netutils.NotInIPList(cluster.GetMaster0IP(), mastersToDelete) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted", cluster.GetMaster0IP())
	}

	// adding custom Env params for delete option here to support executing users clean scripts via env.
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

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
