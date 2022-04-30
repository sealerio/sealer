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

package clusterfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/cert"
	"github.com/sealerio/sealer/pkg/runtime"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
)

const typeV1 = "zlink.aliyun.com/v1alpha1"
const typeV2 = "sealer.cloud/v2"

var ErrClusterNotExist = fmt.Errorf("no cluster exist")

func GetDefaultClusterName() (string, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.sealer", cert.GetUserHomeDir()))
	if err != nil {
		return "", err
	}
	var clusters []string
	for _, f := range files {
		if f.IsDir() {
			clusters = append(clusters, f.Name())
		}
	}
	if len(clusters) == 1 {
		return clusters[0], nil
	} else if len(clusters) > 1 {
		return "", fmt.Errorf("Select a cluster through the -c parameter: " + strings.Join(clusters, ","))
	}

	return "", ErrClusterNotExist
}

func GetClusterFromFile(filepath string) (cluster *v2.Cluster, err error) {
	cluster = &v2.Cluster{}
	if err = utils.UnmarshalYamlFile(filepath, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster from %s, %v", filepath, err)
	}
	cluster.SetAnnotations(common.ClusterfileName, filepath)
	return cluster, nil
}

func GetDefaultCluster() (cluster *v2.Cluster, err error) {
	name, err := GetDefaultClusterName()
	if err != nil {
		return nil, err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var filepath = fmt.Sprintf("%s/.sealer/%s/Clusterfile", userHome, name)

	return GetClusterFromFile(filepath)
}

func GetClusterFromDataCompatV1(data []byte) (*v2.Cluster, error) {
	var cluster *v2.Cluster
	metaType := k8sV1.TypeMeta{}
	err := yaml.Unmarshal(data, &metaType)
	if err != nil {
		return nil, err
	}
	if metaType.Kind != common.Cluster {
		return nil, fmt.Errorf("not found type cluster from: \n%s", data)
	}
	if metaType.APIVersion == typeV1 {
		cluster = &v2.Cluster{}
		clusterV1 := &v1.Cluster{}
		if err := yaml.Unmarshal(data, &clusterV1); err != nil {
			return nil, err
		}
		var hosts []v2.Host
		if len(clusterV1.Spec.Masters.IPList) != 0 {
			hosts = append(hosts, v2.Host{IPS: clusterV1.Spec.Masters.IPList, Roles: []string{common.MASTER}})
		}
		if len(clusterV1.Spec.Nodes.IPList) != 0 {
			hosts = append(hosts, v2.Host{IPS: clusterV1.Spec.Nodes.IPList, Roles: []string{common.NODE}})
		}
		cluster.APIVersion = typeV2
		cluster.Spec.SSH = clusterV1.Spec.SSH
		cluster.Spec.Env = clusterV1.Spec.Env
		cluster.Spec.Hosts = hosts
		cluster.Spec.Image = clusterV1.Spec.Image
		cluster.Name = clusterV1.Name
		cluster.Kind = clusterV1.Kind
	} else {
		c, err := runtime.DecodeCRDFromString(string(data), common.Cluster)
		if err != nil {
			return nil, err
		} else if c == nil {
			return nil, fmt.Errorf("not found type cluster from: \n%s", data)
		}
		cluster = c.(*v2.Cluster)
	}
	return cluster, nil
}
