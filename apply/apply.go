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
	"os"
	"path/filepath"

	k8snet "k8s.io/utils/net"

	"github.com/sealerio/sealer/apply/applydriver"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/store"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
)

type Args struct {
	ClusterName string
	Masters     string
	Nodes       string
	User        string
	Password    string
	Port        uint16
	Pk          string
	PkPassword  string
	PodCidr     string
	SvcCidr     string
	Provider    string
	CustomEnv   []string
	CMDArgs     []string
}

func NewApplierFromFile(path string) (applydriver.Interface, error) {
	if !filepath.IsAbs(path) {
		pa, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(pa, path)
	}
	Clusterfile, err := clusterfile.NewClusterFile(path)
	if err != nil {
		return nil, err
	}

	cluster := Clusterfile.GetCluster()
	if cluster.Name == "" {
		return nil, fmt.Errorf("cluster name cannot be empty, make sure %s file is correct", path)
	}
	if cluster.GetAnnotationsByKey(common.ClusterfileName) == "" {
		cluster.SetAnnotations(common.ClusterfileName, path)
	}

	return NewApplier(&cluster, Clusterfile)
}

func NewApplier(cluster *v2.Cluster, file clusterfile.Interface) (applydriver.Interface, error) {
	if cluster.Name == "" {
		return nil, fmt.Errorf("cluster name cannot be empty")
	}
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	mounter, err := filesystem.NewClusterImageMounter()
	if err != nil {
		return nil, err
	}

	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	hostList := utils.GetIPListFromHosts(cluster.Spec.Hosts)

	if err := checkAllHostsSameFamily(hostList); err != nil {
		return nil, err
	}

	if len(hostList) > 0 && k8snet.IsIPv6String(hostList[0]) &&
		env.ConvertEnv(cluster.Spec.Env)[v2.EnvHostIPFamily] == nil {
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", v2.EnvHostIPFamily, k8snet.IPv6))
	}

	return &applydriver.Applier{
		ClusterDesired:      cluster,
		ClusterFile:         file,
		ImageManager:        imgSvc,
		ClusterImageMounter: mounter,
		ImageStore:          is,
	}, nil
}

func checkAllHostsSameFamily(nodeList []string) error {
	hasIPV4 := false
	hasIPV6 := false
	for _, ip := range nodeList {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return fmt.Errorf("failed to parse %s as a valid ip", ip)
		}
		if k8snet.IsIPv4(parsed) {
			hasIPV4 = true
		} else if k8snet.IsIPv6(parsed) {
			hasIPV6 = true
		}
	}

	if hasIPV4 && hasIPV6 {
		return fmt.Errorf("all hosts must be in same ip family, but the node list given are mixed with ipv4 and ipv6: %v", nodeList)
	}

	return nil
}
