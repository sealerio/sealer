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
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/utils/hash"

	yamlUtils "github.com/sealerio/sealer/utils/yaml"

	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

var ErrClusterNotExist = fmt.Errorf("no cluster exist")

func GetDefaultClusterName() (string, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/.sealer", common.GetHomeDir()))
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
		return "", fmt.Errorf("select a cluster through the -c parameter: " + strings.Join(clusters, ","))
	}

	return "", ErrClusterNotExist
}

func GetClusterFromFile(filepath string) (cluster *v2.Cluster, err error) {
	cluster = &v2.Cluster{}
	if err = yamlUtils.UnmarshalFile(filepath, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster from %s: %v", filepath, err)
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
	return GetClusterFromFile(fmt.Sprintf("%s/.sealer/%s/Clusterfile", userHome, name))
}

// SaveToDisk save cluster obj to disk file with encrypted ssh credential
func SaveToDisk(cluster *v2.Cluster, clusterName string) error {
	fileName := common.GetDefaultClusterfile()
	err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to mkdir %s: %v", fileName, err)
	}

	// if user run cluster image without password,skip to encrypt.
	if !cluster.Spec.SSH.Encrypted && cluster.Spec.SSH.Passwd != "" {
		passwd, err := hash.AesEncrypt([]byte(cluster.Spec.SSH.Passwd))
		if err != nil {
			return err
		}
		cluster.Spec.SSH.Passwd = passwd
		cluster.Spec.SSH.Encrypted = true
	}

	var hosts []v2.Host
	for _, host := range cluster.Spec.Hosts {
		if len(host.IPS) == 0 {
			continue
		}
		if !host.SSH.Encrypted && host.SSH.Passwd != "" {
			passwd, err := hash.AesEncrypt([]byte(host.SSH.Passwd))
			if err != nil {
				return err
			}
			host.SSH.Passwd = passwd
			host.SSH.Encrypted = true
		}
		hosts = append(hosts, host)
	}

	cluster.Spec.Hosts = hosts

	err = yamlUtils.MarshalToFile(fileName, cluster)
	if err != nil {
		return fmt.Errorf("marshal cluster file failed %v", err)
	}
	return nil
}
