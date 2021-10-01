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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/utils"

	"sigs.k8s.io/yaml"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ClusterArgs struct {
	cluster    *v1.Cluster
	imageName  string
	nodeArgs   string
	masterArgs string
	user       string
	passwd     string
	pk         string
	pkPasswd   string
	podCidr    string
	svcCidr    string
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
	c.cluster.Spec.Provider = common.BAREMETAL

	if c.podCidr != "" {
		if flag, err = IsCidrString(c.podCidr); !flag {
			return err
		}
		c.cluster.Spec.Network.PodCIDR = c.podCidr
	}
	if c.svcCidr != "" {
		if flag, err = IsCidrString(c.svcCidr); !flag {
			return err
		}
		c.cluster.Spec.Network.SvcCIDR = c.svcCidr
	}
	if c.passwd != "" {
		c.cluster.Spec.SSH.Passwd = c.passwd
	}
	if IsNumber(c.masterArgs) && (IsNumber(c.nodeArgs) || c.nodeArgs == "") {
		c.cluster.Spec.Masters.Count = c.masterArgs
		c.cluster.Spec.Nodes.Count = c.nodeArgs
		c.cluster.Spec.Provider = common.DefaultCloudProvider
	} else if IsIPList(c.masterArgs) && (IsIPList(c.nodeArgs) || c.nodeArgs == "") {
		c.cluster.Spec.Masters.IPList = strings.Split(c.masterArgs, ",")
		if c.nodeArgs != "" {
			c.cluster.Spec.Nodes.IPList = strings.Split(c.nodeArgs, ",")
		}
		c.cluster.Spec.SSH.User = c.user
		c.cluster.Spec.SSH.Pk = c.pk
		c.cluster.Spec.SSH.PkPasswd = c.pkPasswd
	} else {
		err = fmt.Errorf("enter true iplist or count")
	}

	return err
}

func GetClusterFileByImageName(imageName string) (cluster *v1.Cluster, err error) {
	clusterFile, err := image.GetClusterFileFromImageManifest(imageName)
	fmt.Println(clusterFile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(clusterFile), &cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func NewApplierFromArgs(imageName string, runArgs *common.RunArgs) (Interface, error) {
	cluster, err := GetClusterFileByImageName(imageName)
	if err != nil {
		return nil, err
	}
	if runArgs.Nodes == "" && runArgs.Masters == "" {
		return NewApplier(cluster)
	}
	c := &ClusterArgs{
		cluster:    cluster,
		imageName:  imageName,
		nodeArgs:   runArgs.Nodes,
		masterArgs: runArgs.Masters,
		user:       runArgs.User,
		passwd:     runArgs.Password,
		pk:         runArgs.Pk,
		pkPasswd:   runArgs.PkPassword,
		podCidr:    runArgs.PodCidr,
		svcCidr:    runArgs.SvcCidr,
	}
	if err := c.SetClusterArgs(); err != nil {
		return nil, err
	}
	return NewApplier(c.cluster)
}
