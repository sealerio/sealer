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

	"github.com/alibaba/sealer/apply/applytype"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type ClusterArgs struct {
	cluster   *v1.Cluster
	imageName string
	runArgs   *common.RunArgs
}

func IsNumber(args string) bool {
	_, err := strconv.Atoi(args)
	return err == nil
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

func IsCidrString(arg string) (bool, error) {
	_, err := utils.ParseCIDR(arg)
	var flag bool
	if err == nil {
		flag = true
	}
	return flag, err
}

func (c *ClusterArgs) SetClusterArgs() error {
	var err error = nil
	var flag bool
	c.cluster.Spec.Image = c.imageName

	if c.runArgs.PodCidr != "" {
		if flag, err = IsCidrString(c.runArgs.PodCidr); !flag {
			return err
		}
		c.cluster.Spec.Network.PodCIDR = c.runArgs.PodCidr
	}
	if c.runArgs.SvcCidr != "" {
		if flag, err = IsCidrString(c.runArgs.SvcCidr); !flag {
			return err
		}
		c.cluster.Spec.Network.SvcCIDR = c.runArgs.SvcCidr
	}
	if c.runArgs.Password != "" {
		c.cluster.Spec.SSH.Passwd = c.runArgs.Password
	}
	if IsNumber(c.runArgs.Masters) && (IsNumber(c.runArgs.Nodes) || c.runArgs.Nodes == "") {
		c.cluster.Spec.Masters.Count = c.runArgs.Masters
		c.cluster.Spec.Nodes.Count = c.runArgs.Nodes
		if c.runArgs.Nodes == "" {
			c.cluster.Spec.Nodes.Count = "0"
		}
		if c.runArgs.Provider != "" {
			c.cluster.Spec.Provider = c.runArgs.Provider
			if !utils.InList(c.runArgs.Provider, []string{common.AliCloud, common.CONTAINER}) {
				return fmt.Errorf("the provider cannot be set to %s", c.runArgs.Provider)
			}
		}
	} else if IsIPList(c.runArgs.Masters) && (IsIPList(c.runArgs.Nodes) || c.runArgs.Nodes == "") {
		c.cluster.Spec.Masters.IPList = strings.Split(c.runArgs.Masters, ",")
		if c.runArgs.Nodes != "" {
			c.cluster.Spec.Nodes.IPList = strings.Split(c.runArgs.Nodes, ",")
		}
		c.cluster.Spec.SSH.User = c.runArgs.User
		c.cluster.Spec.SSH.Pk = c.runArgs.Pk
		c.cluster.Spec.SSH.PkPasswd = c.runArgs.PkPassword
		c.cluster.Spec.Provider = common.BAREMETAL
	} else {
		err = fmt.Errorf("enter true iplist or count")
	}

	return err
}

func GetClusterFileByImageName(imageName string) (cluster *v1.Cluster, err error) {
	clusterFile, err := image.GetClusterFileFromImageManifest(imageName)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(clusterFile), &cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func NewApplierFromArgs(imageName string, runArgs *common.RunArgs) (applytype.Interface, error) {
	cluster, err := GetClusterFileByImageName(imageName)
	if err != nil {
		return nil, err
	}
	if runArgs.Nodes == "" && runArgs.Masters == "" {
		return NewApplier(cluster)
	}
	c := &ClusterArgs{
		cluster:   cluster,
		imageName: imageName,
		runArgs:   runArgs,
	}
	if err := c.SetClusterArgs(); err != nil {
		return nil, err
	}
	return NewApplier(c.cluster)
}
