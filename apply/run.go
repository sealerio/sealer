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
	"net"
	"strconv"
	"strings"

	v1 "github.com/alibaba/sealer/types/api/v1"

	"github.com/alibaba/sealer/apply/applydriver"

	"github.com/alibaba/sealer/common"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

type ClusterArgs struct {
	cluster   *v2.Cluster
	imageName string
	runArgs   *common.RunArgs
	hosts     []v2.Host
}

func IsIPList(args string) bool {
	ipList := strings.Split(args, ",")

	for _, i := range ipList {
		if !strings.Contains(i, ":") {
			return net.ParseIP(i) != nil
		}
		if _, err := net.ResolveTCPAddr("tcp", i); err != nil {
			return false
		}
	}
	return true
}

func PreProcessIPList(joinArgs *common.RunArgs) error {
	if err := utils.AssemblyIPList(&joinArgs.Masters); err != nil {
		return err
	}
	if err := utils.AssemblyIPList(&joinArgs.Nodes); err != nil {
		return err
	}
	return nil
}

func (c *ClusterArgs) SetClusterArgs() error {
	c.cluster.APIVersion = common.APIVersion
	c.cluster.Kind = common.Cluster
	c.cluster.Name = c.runArgs.ClusterName
	c.cluster.Spec.Image = c.imageName
	c.cluster.Spec.SSH.User = c.runArgs.User
	c.cluster.Spec.SSH.Pk = c.runArgs.Pk
	c.cluster.Spec.SSH.PkPasswd = c.runArgs.PkPassword
	c.cluster.Spec.SSH.Port = strconv.Itoa(int(c.runArgs.Port))
	c.cluster.Spec.Env = append(c.cluster.Spec.Env, c.runArgs.CustomEnv...)
	c.cluster.Spec.CMDArgs = append(c.cluster.Spec.CMDArgs, c.runArgs.CMDArgs...)
	if c.runArgs.Password != "" {
		c.cluster.Spec.SSH.Passwd = c.runArgs.Password
	}
	err := PreProcessIPList(c.runArgs)
	if err != nil {
		return err
	}
	if IsIPList(c.runArgs.Masters) && (IsIPList(c.runArgs.Nodes) || c.runArgs.Nodes == "") {
		masters := strings.Split(c.runArgs.Masters, ",")
		nodes := strings.Split(c.runArgs.Nodes, ",")
		c.hosts = []v2.Host{}
		c.setHostWithIpsPort(masters, common.MASTER)
		if len(nodes) != 0 {
			c.setHostWithIpsPort(nodes, common.NODE)
		}
		c.cluster.Spec.Hosts = c.hosts
	} else {
		err = fmt.Errorf("enter true iplist or count")
	}

	return err
}

func (c *ClusterArgs) setHostWithIpsPort(ips []string, role string) {
	//map[ssh port]*host
	hostMap := map[string]*v2.Host{}
	for i := range ips {
		ip, port := utils.GetHostIPAndPortOrDefault(ips[i], strconv.Itoa(int(c.runArgs.Port)))
		if _, ok := hostMap[port]; !ok {
			hostMap[port] = &v2.Host{IPS: []string{ip}, Roles: []string{role}, SSH: v1.SSH{Port: port}}
			continue
		}
		hostMap[port].IPS = append(hostMap[port].IPS, ip)
	}
	_, master0Port := utils.GetHostIPAndPortOrDefault(ips[0], strconv.Itoa(int(c.runArgs.Port)))
	for port, host := range hostMap {
		host.IPS = removeIPListDuplicatesAndEmpty(host.IPS)
		if port == master0Port && role == common.MASTER {
			c.hosts = append([]v2.Host{*host}, c.hosts...)
			continue
		}
		c.hosts = append(c.hosts, *host)
	}
}

func NewApplierFromArgs(imageName string, runArgs *common.RunArgs) (applydriver.Interface, error) {
	c := &ClusterArgs{
		cluster:   &v2.Cluster{},
		imageName: imageName,
		runArgs:   runArgs,
	}
	if err := c.SetClusterArgs(); err != nil {
		return nil, err
	}
	return NewApplier(c.cluster)
}
