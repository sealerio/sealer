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

	"github.com/alibaba/sealer/pkg/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/apply/v2/applydriver"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

const typeV1 = "zlink.aliyun.com/v1alpha1"
const typeV2 = "sealer.cloud/v2"

type ClusterArgs struct {
	cluster   *v2.Cluster
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

func (c *ClusterArgs) SetClusterArgs() error {
	var err error = nil
	c.cluster.Spec.Image = c.imageName
	c.cluster.Spec.SSH.User = c.runArgs.User
	c.cluster.Spec.SSH.Pk = c.runArgs.Pk
	c.cluster.Spec.SSH.PkPasswd = c.runArgs.PkPassword
	if c.runArgs.CustomEnv != nil {
		c.cluster.Spec.Env = c.runArgs.CustomEnv
	}
	if c.runArgs.Password != "" {
		c.cluster.Spec.SSH.Passwd = c.runArgs.Password
	}
	if IsIPList(c.runArgs.Masters) && (IsIPList(c.runArgs.Nodes) || c.runArgs.Nodes == "") {
		var hosts []v2.Host
		hosts = append(hosts, v2.Host{IPS: strings.Split(c.runArgs.Masters, ","), Roles: []string{common.MASTER}})
		if c.runArgs.Nodes != "" {
			hosts = append(hosts, v2.Host{IPS: strings.Split(c.runArgs.Nodes, ","), Roles: []string{common.NODE}})
		}
		c.cluster.Spec.Hosts = hosts
	} else {
		err = fmt.Errorf("enter true iplist or count")
	}

	return err
}

func GetClusterFileByImageName(imageName string) (*v2.Cluster, error) {
	clusterFile, err := image.GetClusterFileFromImageManifest(imageName)
	if err != nil {
		return nil, err
	}
	return GetClusterFromDataCompatV1(clusterFile)
}

func GetClusterFromDataCompatV1(data string) (*v2.Cluster, error) {
	cluster := &v2.Cluster{}
	metaType := metav1.TypeMeta{}
	err := yaml.Unmarshal([]byte(data), &metaType)
	if err != nil {
		return nil, fmt.Errorf("decode cluster failed %v", err)
	}
	if metaType.APIVersion == typeV1 {
		c1 := &v1.Cluster{}
		if err := yaml.Unmarshal([]byte(data), &c1); err != nil {
			return nil, err
		}
		var hosts []v2.Host
		if len(c1.Spec.Masters.IPList) != 0 {
			hosts = append(hosts, v2.Host{IPS: c1.Spec.Masters.IPList, Roles: []string{common.MASTER}})
		}
		if len(c1.Spec.Nodes.IPList) != 0 {
			hosts = append(hosts, v2.Host{IPS: c1.Spec.Nodes.IPList, Roles: []string{common.NODE}})
		}
		cluster.APIVersion = typeV2
		cluster.Spec.SSH = c1.Spec.SSH
		cluster.Spec.Env = c1.Spec.Env
		cluster.Spec.Hosts = hosts
		cluster.Spec.Image = c1.Spec.Image
		cluster.Name = c1.Name
		cluster.Kind = c1.Kind
	} else {
		c, err := runtime.DecodeCRDFromString(data, common.Cluster)
		if err != nil {
			return nil, err
		} else if c == nil {
			return nil, fmt.Errorf("not found type cluster from %s", data)
		}
		cluster = c.(*v2.Cluster)
	}
	return cluster, nil
}

func NewApplierFromArgs(imageName string, runArgs *common.RunArgs) (applydriver.Interface, error) {
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
