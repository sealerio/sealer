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

	"github.com/sealerio/sealer/apply/applydriver"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/hash"
	"github.com/sealerio/sealer/utils/net"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sealerio/sealer/utils/yaml"
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

	applier, err := NewDefaultApplier(cluster, nil)
	if err != nil {
		return nil, err
	}
	return applier, nil
}

func Join(cluster *v2.Cluster, scaleArgs *Args) error {
	return joinBaremetalNodes(cluster, scaleArgs)
}

func joinBaremetalNodes(cluster *v2.Cluster, scaleArgs *Args) error {
	var err error
	// merge custom Env to the existed cluster
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	if err = PreProcessIPList(scaleArgs); err != nil {
		return err
	}

	if (!net.IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!net.IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf("parameter error: The current mode should submit iplist")
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

	masterIPs := cluster.GetMasterIPList()
	workerIPs := cluster.GetNodeIPList()

	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}

	nodeListFromK8s, err := client.ListNodes()
	if err != nil {
		return err
	}

	var mastersFromK8s, workersFromK8s []string
	for i, n := range nodeListFromK8s.Items {
		if _, ok := n.Labels[applydriver.MasterRoleLabel]; ok {
			mastersFromK8s = append(mastersFromK8s, k8s.GetInternalIP(&nodeListFromK8s.Items[i]))
		} else {
			workersFromK8s = append(workersFromK8s, k8s.GetInternalIP(&nodeListFromK8s.Items[i]))
		}
	}

	a, b := strUtils.Diff(masterIPs, mastersFromK8s)
	if len(a) != 0 || len(b) != 0 {
		return fmt.Errorf("we find current clusterfile's master ip list is different from actual k8s's, please check")
	}
	a, b = strUtils.Diff(workerIPs, workersFromK8s)
	if len(a) != 0 || len(b) != 0 {
		return fmt.Errorf("we find current clusterfile's worker ip list is different from actual k8s's, please check")
	}

	//add joined masters
	if scaleArgs.Masters != "" {
		toAdd := removeDuplicate(strings.Split(scaleArgs.Masters, ","))

		// skip already added masters
		for _, ip := range masterIPs {
			toAdd = strUtils.RemoveString(toAdd, ip, nil)
		}

		host := v2.Host{
			IPS:   toAdd,
			Roles: []string{common.MASTER},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}

	//add joined nodes
	if scaleArgs.Nodes != "" {
		toAdd := removeDuplicate(strings.Split(scaleArgs.Nodes, ","))

		// skip already added workers
		for _, ip := range workerIPs {
			toAdd = strUtils.RemoveString(toAdd, ip, nil)
		}

		host := v2.Host{
			IPS:   toAdd,
			Roles: []string{common.NODE},
		}

		if changedSSH != nil {
			host.SSH = *changedSSH
		}

		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}

	return nil
}

func removeDuplicate(ipList []string) []string {
	return strUtils.RemoveDuplicate(strUtils.NewComparator(ipList, []string{""}).GetSrcSubtraction())
}

func Delete(cluster *v2.Cluster, scaleArgs *Args) error {
	return deleteBaremetalNodes(cluster, scaleArgs)
}

func deleteBaremetalNodes(cluster *v2.Cluster, scaleArgs *Args) error {
	// adding custom Env params for delete option here to support executing users clean scripts via env.
	cluster.Spec.Env = append(cluster.Spec.Env, scaleArgs.CustomEnv...)

	if err := PreProcessIPList(scaleArgs); err != nil {
		return err
	}

	if (!net.IsIPList(scaleArgs.Nodes) && scaleArgs.Nodes != "") || (!net.IsIPList(scaleArgs.Masters) && scaleArgs.Masters != "") {
		return fmt.Errorf("parameter error: The current mode should submit iplist")
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
