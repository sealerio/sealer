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
	"strings"

	"github.com/sealerio/sealer/utils/hash"
	netutils "github.com/sealerio/sealer/utils/net"
	strUtils "github.com/sealerio/sealer/utils/strings"

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func ConstructClusterFromArg(imageName string, runArgs *apply.Args) (*v2.Cluster, error) {
	resultHosts, err := GetHosts(runArgs.Masters, runArgs.Nodes)
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

func removeDuplicate(ipList []string) []string {
	return strUtils.RemoveDuplicate(strUtils.NewComparator(ipList, []string{""}).GetSrcSubtraction())
}

func JoinClusterNode(cluster *v2.Cluster, scaleArgs *apply.Args, joinMasters, joinWorkers string) error {
	return joinBaremetalNodes(cluster, scaleArgs, joinMasters, joinWorkers)
}

func joinBaremetalNodes(cluster *v2.Cluster, scaleArgs *apply.Args, joinMasters, joinWorkers string) error {
	var err error
	// merge custom Env to the existed cluster
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	joinMasters, err = netutils.AssemblyIPList(joinMasters)
	if err != nil {
		return err
	}

	joinWorkers, err = netutils.AssemblyIPList(joinWorkers)
	if err != nil {
		return err
	}

	if (!netutils.IsIPList(joinWorkers) && joinWorkers != "") || (!netutils.IsIPList(joinMasters) && joinMasters != "") {
		return fmt.Errorf("parameter error: current mode should submit iplist")
	}

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
	if joinMasters != "" {
		masterIPs := cluster.GetMasterIPList()
		addedMasterIPStr := removeDuplicate(strings.Split(joinMasters, ","))
		addedMasterIP := netutils.IPStrsToIPs(addedMasterIPStr)

		for _, ip := range addedMasterIP {
			// if ip already taken by master will return join duplicated ip error
			if !netutils.NotInIPList(ip, masterIPs) {
				return fmt.Errorf("failed to scale master for duplicated ip: %s", ip)
			}
		}

		host := v2.Host{
			IPS:   addedMasterIP,
			Roles: []string{common.MASTER},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}

	//add joined nodes
	if joinWorkers != "" {
		nodeIPs := cluster.GetNodeIPList()
		addedNodeIPStrs := removeDuplicate(strings.Split(joinWorkers, ","))
		addedNodeIP := netutils.IPStrsToIPs(addedNodeIPStrs)

		for _, ip := range addedNodeIP {
			// if ip already taken by node will return join duplicated ip error
			if !netutils.NotInIPList(ip, nodeIPs) {
				return fmt.Errorf("failed to scale node for duplicated ip: %s", ip)
			}
		}

		host := v2.Host{
			IPS:   addedNodeIP,
			Roles: []string{common.NODE},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}
	return nil
}

func DeleteClusterNode(cluster *v2.Cluster, scaleArgs *apply.Args, deleteMasters, deleteWorkers string) error {
	return deleteBaremetalNodes(cluster, scaleArgs, deleteMasters, deleteWorkers)
}

func deleteBaremetalNodes(cluster *v2.Cluster, scaleArgs *apply.Args, mastersToDelete, workersToDelete string) error {
	var err error
	// adding custom Env params for delete option here to support executing users clean scripts via env.
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	mastersToDelete, err = netutils.AssemblyIPList(mastersToDelete)
	if err != nil {
		return err
	}

	workersToDelete, err = netutils.AssemblyIPList(workersToDelete)
	if err != nil {
		return err
	}

	if (!netutils.IsIPList(workersToDelete) && workersToDelete != "") || (!netutils.IsIPList(mastersToDelete) && mastersToDelete != "") {
		return fmt.Errorf("parameter error: current mode should submit iplist")
	}

	//master0 machine cannot be deleted
	scaleMasterIPs := netutils.IPStrsToIPs(strings.Split(mastersToDelete, ","))
	if !netutils.NotInIPList(cluster.GetMaster0IP(), scaleMasterIPs) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted", cluster.GetMaster0IP())
	}

	if mastersToDelete != "" && netutils.IsIPList(mastersToDelete) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				masterIPs := netutils.IPStrsToIPs(strings.Split(mastersToDelete, ","))
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, masterIPs)
			}
		}
	}
	if workersToDelete != "" && netutils.IsIPList(workersToDelete) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.NODE, cluster.Spec.Hosts[i].Roles) {
				nodeIPs := netutils.IPStrsToIPs(strings.Split(workersToDelete, ","))
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, nodeIPs)
			}
		}
	}
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
