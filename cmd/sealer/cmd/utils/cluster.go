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
	"reflect"
	"strconv"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/types/api/constants"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/maps"
	netutils "github.com/sealerio/sealer/utils/net"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// MergeClusterWithImageExtension :set default value get from image extension,such as image global env
func MergeClusterWithImageExtension(cluster *v2.Cluster, imageExt imagev1.ImageExtension) *v2.Cluster {
	if len(imageExt.Env) > 0 {
		envs := maps.ConvertToSlice(imageExt.Env)
		envs = append(envs, cluster.Spec.Env...)
		cluster.Spec.Env = envs
	}

	return cluster
}

func MergeClusterWithFlags(cluster v2.Cluster, mergeFlags *types.MergeFlags) (*v2.Cluster, error) {
	if len(mergeFlags.CustomEnv) > 0 {
		cluster.Spec.Env = append(cluster.Spec.Env, mergeFlags.CustomEnv...)
	}

	if len(mergeFlags.Cmds) > 0 {
		cluster.Spec.CMD = mergeFlags.Cmds
	}

	if len(mergeFlags.AppNames) > 0 {
		cluster.Spec.APPNames = mergeFlags.AppNames
	}

	// if no master and node specify form flag, just return.
	if len(mergeFlags.Masters) == 0 && len(mergeFlags.Nodes) == 0 {
		return &cluster, nil
	}

	flagMasters, flagNodes, err := ParseToNetIPList(mergeFlags.Masters, mergeFlags.Nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	//validate run flags masters
	masterIPs := cluster.GetMasterIPList()
	for _, ip := range flagMasters {
		if netutils.IsInIPList(ip, masterIPs) {
			return nil, fmt.Errorf("failed to merge master ip form flags, duplicated ip is: %s", ip)
		}
	}

	//validate run flags nodes
	nodeIPs := cluster.GetNodeIPList()
	for _, ip := range flagNodes {
		if netutils.IsInIPList(ip, nodeIPs) {
			return nil, fmt.Errorf("failed to merge node ip form flags, duplicated ip is: %s", ip)
		}
	}

	//TODO: validate ssh auth
	flagHosts := TransferIPToHosts(flagMasters, flagNodes, v1.SSH{
		User:     mergeFlags.User,
		Passwd:   mergeFlags.Password,
		PkPasswd: mergeFlags.PkPassword,
		Pk:       mergeFlags.Pk,
		Port:     strconv.Itoa(int(mergeFlags.Port)),
	})

	cluster.Spec.Hosts = append(cluster.Spec.Hosts, flagHosts...)
	return &cluster, err
}

func ConstructClusterForRun(imageName string, runFlags *types.RunFlags) (*v2.Cluster, error) {
	masterIPList, nodeIPList, err := ParseToNetIPList(runFlags.Masters, runFlags.Nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			SSH: v1.SSH{
				User:     runFlags.User,
				Passwd:   runFlags.Password,
				PkPasswd: runFlags.PkPassword,
				Pk:       runFlags.Pk,
				Port:     strconv.Itoa(int(runFlags.Port)),
			},
			Image: imageName,
			//use cluster ssh auth by default
			Hosts:    TransferIPToHosts(masterIPList, nodeIPList, v1.SSH{}),
			Env:      runFlags.CustomEnv,
			CMD:      runFlags.Cmds,
			APPNames: runFlags.AppNames,
		},
	}
	cluster.APIVersion = v2.GroupVersion.String()
	cluster.Kind = constants.ClusterKind
	cluster.Name = "my-cluster"
	return &cluster, nil
}

func ConstructClusterForScaleUp(cluster *v2.Cluster, scaleFlags *types.ScaleUpFlags, currentNodes, joinMasters, joinWorkers []net.IP) (mj, nj []net.IP, err error) {
	mj, _ = strUtils.Diff(currentNodes, joinMasters)
	nj, _ = strUtils.Diff(currentNodes, joinWorkers)

	if len(mj) == 0 && len(nj) == 0 {
		return nil, nil, fmt.Errorf("scale ip %v is already in the current cluster %v", append(joinMasters, joinWorkers...), currentNodes)
	}

	nodes := cluster.GetAllIPList()
	//TODO Add password encryption mode in the future
	//add joined masters
	for _, ip := range mj {
		// if ip already taken by node, skip it
		if netutils.IsInIPList(ip, nodes) {
			return nil, nil, fmt.Errorf("failed to scale master for duplicated ip: %s", ip)
		}
	}
	if len(mj) != 0 {
		host := constructHost(common.MASTER, mj, scaleFlags, cluster.Spec.SSH)
		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}

	for _, ip := range nj {
		// if ip already taken by node, skip it
		if netutils.IsInIPList(ip, nodes) {
			return nil, nil, fmt.Errorf("failed to scale node for duplicated ip: %s", ip)
		}
	}
	//add joined nodes
	if len(nj) != 0 {
		host := constructHost(common.NODE, nj, scaleFlags, cluster.Spec.SSH)
		cluster.Spec.Hosts = append(cluster.Spec.Hosts, host)
	}

	return mj, nj, nil
}

func ConstructClusterForScaleDown(cluster *v2.Cluster, mastersToDelete, workersToDelete []net.IP) error {
	if len(mastersToDelete) != 0 {
		for i := range cluster.Spec.Hosts {
			if strUtils.IsInSlice(common.MASTER, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = netutils.RemoveIPs(cluster.Spec.Hosts[i].IPS, mastersToDelete)
			}
			continue
		}
	}

	if len(workersToDelete) != 0 {
		for i := range cluster.Spec.Hosts {
			if strUtils.IsInSlice(common.NODE, cluster.Spec.Hosts[i].Roles) {
				cluster.Spec.Hosts[i].IPS = netutils.RemoveIPs(cluster.Spec.Hosts[i].IPS, workersToDelete)
			}
			continue
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

func constructHost(role string, joinIPs []net.IP, scaleFlags *types.ScaleUpFlags, clusterSSH v1.SSH) v2.Host {
	//todo we could support host level env form cli later.
	//todo we could support host level role form cli later.
	host := v2.Host{
		IPS:   joinIPs,
		Roles: []string{role},
		Env:   scaleFlags.CustomEnv,
	}

	scaleFlagSSH := v1.SSH{
		User:     scaleFlags.User,
		Passwd:   scaleFlags.Password,
		Port:     strconv.Itoa(int(scaleFlags.Port)),
		Pk:       scaleFlags.Pk,
		PkPasswd: scaleFlags.PkPassword,
	}

	if reflect.DeepEqual(scaleFlagSSH, clusterSSH) {
		return host
	}

	host.SSH = scaleFlagSSH
	return host
}

func GetCurrentCluster(client *k8s.Client) (*v2.Cluster, error) {
	nodes, err := client.ListNodes()
	if err != nil {
		return nil, err
	}

	cluster := &v2.Cluster{}
	var masterIPList []net.IP
	var nodeIPList []net.IP

	for _, node := range nodes.Items {
		addr := getNodeAddress(node)
		if addr == nil {
			return nil, fmt.Errorf("failed to get node address for node %s", node.Name)
		}
		if _, ok := node.Labels[common.MasterRoleLabel]; ok {
			masterIPList = append(masterIPList, addr)
			continue
		}
		nodeIPList = append(nodeIPList, addr)
	}
	cluster.Spec.Hosts = []v2.Host{{IPS: masterIPList, Roles: []string{common.MASTER}}, {IPS: nodeIPList, Roles: []string{common.NODE}}}

	return cluster, nil
}

func getNodeAddress(node corev1.Node) net.IP {
	if len(node.Status.Addresses) < 1 {
		return nil
	}

	var IP string
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			IP = address.Address
			break
		}
	}

	return net.ParseIP(IP)
}

func GetClusterClient() *k8s.Client {
	client, err := k8s.NewK8sClient()
	if client != nil {
		return client
	}
	if err != nil {
		logrus.Warnf("try to new k8s client via default kubeconfig, maybe this is a new cluster that needs to be created: %v", err)
	}
	return nil
}
