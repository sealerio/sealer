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

package upgrade

import (
	"strings"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	upgradeFirstMaster(client ssh.Client, IP, version string)
	upgradeOtherMaster(client ssh.Client, IP, version string)
	upgradeNode(client ssh.Client, IP, version string)
}

func ClusterUpgrade(version string, args common.UpgradeArgs) error {
	//TODO 判断是否是一个可升级的版本（有这个版本，且是新版本）
	//获取当前集群的各个节点的IP地址和ssh密码（root用户）//从输入参数中进行获取
	cluster, _ := NewUpgrader(args)
	client, _ := ssh.NewSSHClientWithCluster(&cluster)
	var debian debianDistribution
	var redhat redhatDistribution
	//第一个控制节点上的操作
	if client.SSH.IsFileExist(client.Host, "/etc/debian_version") {
		debian.upgradeFirstMaster(client, client.Host, version)
	} else if client.SSH.IsFileExist(client.Host, "/etc/redhat-release") {
		redhat.upgradeFirstMaster(client, client.Host, version)
	}
	//其余控制节点的操作
	for _, IP := range cluster.Spec.Masters.IPList {
		if client.SSH.IsFileExist(IP, "/etc/debian_version") {
			debian.upgradeOtherMaster(client, IP, version)
		} else if client.SSH.IsFileExist(IP, "/etc/redhat-release") {
			redhat.upgradeOtherMaster(client, IP, version)
		}
	}
	//工作节点的操作
	for _, IP := range cluster.Spec.Nodes.IPList {
		if client.SSH.IsFileExist(IP, "/etc/debian_version") {
			debian.upgradeNode(client, IP, version)
		} else if client.SSH.IsFileExist(IP, "/etc/redhat-release") {
			redhat.upgradeNode(client, IP, version)
		}
	}
	return nil
}

func NewUpgrader(args common.UpgradeArgs) (v1.Cluster, error) {
	var cluster v1.Cluster
	//optional TODO:ssh登陆到集群中的一个节点上，获取Clusterfile文件

	if args.Masters != "" {
		cluster.Spec.Masters.IPList = strings.Split(args.Masters, ",")
	} else {
		cluster.Spec.Nodes.IPList = nil
	}

	if args.Nodes != "" {
		cluster.Spec.Nodes.IPList = strings.Split(args.Nodes, ",")
	} else {
		cluster.Spec.Nodes.IPList = nil
	}
	//cluster.Spec.Provider = common.BAREMETAL
	cluster.Spec.SSH = v1.SSH{
		User:   common.ROOT,
		Passwd: args.Passwd,
	}
	return cluster, nil
}
