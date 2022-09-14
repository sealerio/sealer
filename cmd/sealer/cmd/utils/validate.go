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

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/hash"
	strUtils "github.com/sealerio/sealer/utils/strings"

	"github.com/sealerio/sealer/apply"

	netutils "github.com/sealerio/sealer/utils/net"
)

// ValidateRunArgs validates all the input args from sealer run command.
func ValidateRunArgs(runArgs *apply.Args) error {
	// TODO: add detailed validation steps.
	var errMsg []string

	// validate input masters IP info
	if err := ValidateIPStr(runArgs.Masters); err != nil {
		errMsg = append(errMsg, err.Error())
	}

	// validate input nodes IP info
	if len(runArgs.Nodes) != 0 {
		// empty runArgs.Nodes are valid, since no nodes are input.
		if err := ValidateIPStr(runArgs.Nodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}

func ValidateIPStr(inputStr string) error {
	if len(inputStr) == 0 {
		return fmt.Errorf("input IP info cannot be empty")
	}

	// 1. validate if it is IP range
	if strings.Contains(inputStr, "-") {
		ips := strings.Split(inputStr, "-")
		if len(ips) != 2 {
			return fmt.Errorf("input IP(%s) is range format but invalid, IP range format must be xxx.xxx.xxx.1-xxx.xxx.xxx.70", inputStr)
		}

		if net.ParseIP(ips[0]) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ips[0])
		}
		if net.ParseIP(ips[1]) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ips[1])
		}

		if netutils.CompareIP(ips[0], ips[1]) >= 0 {
			return fmt.Errorf("input IP(%s) must be less than input IP(%s)", ips[0], ips[1])
		}

		return nil
	}

	// 2. validate if it is IP list, like 192.168.0.5,192.168.0.6,192.168.0.7
	for _, ip := range strings.Split(inputStr, ",") {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ip)
		}
	}

	return nil
}

// ValidateJoinArgs validates all the input args from sealer join command.
func ValidateJoinArgs(joinMasters, joinNodes string) error {
	var errMsg []string

	if joinNodes == "" && joinMasters == "" {
		return fmt.Errorf("master and node cannot both be empty")
	}

	// validate input masters IP info
	if len(joinMasters) != 0 {
		if err := ValidateIPStr(joinMasters); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	// validate input nodes IP info
	if len(joinNodes) != 0 {
		if err := ValidateIPStr(joinNodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}

func removeDuplicate(ipList []string) []string {
	return strUtils.RemoveDuplicate(strUtils.NewComparator(ipList, []string{""}).GetSrcSubtraction())
}

func Join(cluster *v2.Cluster, scaleArgs *apply.Args, joinMasters, joinWorkers string) error {
	return joinBaremetalNodes(cluster, scaleArgs, joinMasters, joinWorkers)
}

func joinBaremetalNodes(cluster *v2.Cluster, scaleArgs *apply.Args, joinMasters, joinWorkers string) error {
	var err error
	// merge custom Env to the existed cluster
	//cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

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

func Delete(cluster *v2.Cluster, deleteMasters, deleteWorkers string) error {
	return deleteBaremetalNodes(cluster, deleteMasters, deleteWorkers)
}

func deleteBaremetalNodes(cluster *v2.Cluster, deleteMasters, deleteWorkers string) error {
	var err error
	// adding custom Env params for delete option here to support executing users clean scripts via env.
	//cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	deleteMasters, err = netutils.AssemblyIPList(deleteMasters)
	if err != nil {
		return err
	}

	deleteWorkers, err = netutils.AssemblyIPList(deleteWorkers)
	if err != nil {
		return err
	}

	if (!netutils.IsIPList(deleteWorkers) && deleteWorkers != "") || (!netutils.IsIPList(deleteMasters) && deleteMasters != "") {
		return fmt.Errorf("parameter error: current mode should submit iplist")
	}

	//master0 machine cannot be deleted
	scaleMasterIPs := netutils.IPStrsToIPs(strings.Split(deleteMasters, ","))
	if !netutils.NotInIPList(cluster.GetMaster0IP(), scaleMasterIPs) {
		return fmt.Errorf("master0 machine(%s) cannot be deleted", cluster.GetMaster0IP())
	}

	if deleteMasters != "" && netutils.IsIPList(deleteMasters) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				masterIPs := netutils.IPStrsToIPs(strings.Split(deleteMasters, ","))
				cluster.Spec.Hosts[i].IPS = returnFilteredIPList(cluster.Spec.Hosts[i].IPS, masterIPs)
			}
		}
	}
	if deleteWorkers != "" && netutils.IsIPList(deleteWorkers) {
		for i := range cluster.Spec.Hosts {
			if !strUtils.NotIn(common.NODE, cluster.Spec.Hosts[i].Roles) {
				nodeIPs := netutils.IPStrsToIPs(strings.Split(deleteWorkers, ","))
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
