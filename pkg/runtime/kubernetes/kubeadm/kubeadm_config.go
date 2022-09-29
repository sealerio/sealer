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

package kubeadm

import (
	"fmt"
	"io"

	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"

	"github.com/sealerio/sealer/utils"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/strings"

	"github.com/imdario/mergo"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"
)

// Read config from https://github.com/sealerio/sealer/blob/main/docs/design/clusterfile-v2.md and overwrite default kubeadm.yaml
// Use github.com/imdario/mergo to merge kubeadm config in Clusterfile and the default kubeadm config
// Using a config filter to handle some edge cases

// https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/apis/kubeadm/v1beta2/types.go
// Using map to overwrite Kubeadm configs

// nolint
type KubeadmConfig struct {
	v1beta2.InitConfiguration
	v1beta2.ClusterConfiguration
	v1alpha1.KubeProxyConfiguration
	v1beta1.KubeletConfiguration
	v1beta2.JoinConfiguration
}

// LoadFromClusterfile :Load KubeadmConfig from Clusterfile.
// If it has `KubeadmConfig` in Clusterfile, load every field to each configuration.
// If Kubeadm raw config in Clusterfile, just load it.
func (k *KubeadmConfig) LoadFromClusterfile(kubeadmConfig *KubeadmConfig) error {
	if kubeadmConfig == nil {
		return nil
	}
	k.APIServer.CertSANs = strings.RemoveDuplicate(append(k.APIServer.CertSANs, kubeadmConfig.APIServer.CertSANs...))
	return mergo.Merge(k, kubeadmConfig)
}

// Merge Using github.com/imdario/mergo to merge KubeadmConfig to the ClusterImage default kubeadm config, overwrite some field.
// if defaultKubeadmConfig file not exist, use default raw kubeadm config to merge k.KubeConfigSpec empty value
func (k *KubeadmConfig) Merge(kubeadmYamlPath string) error {
	var (
		defaultKubeadmConfig *KubeadmConfig
		err                  error
	)
	if kubeadmYamlPath == "" || !osi.IsFileExist(kubeadmYamlPath) {
		defaultKubeadmConfig, err = LoadKubeadmConfigs(DefaultKubeadmConfig, utils.DecodeCRDFromString)
		if err != nil {
			return err
		}
		return mergo.Merge(k, defaultKubeadmConfig)
	}
	defaultKubeadmConfig, err = LoadKubeadmConfigs(kubeadmYamlPath, utils.DecodeCRDFromFile)
	if err != nil {
		return fmt.Errorf("failed to found kubeadm config from %s: %v", kubeadmYamlPath, err)
	}
	k.APIServer.CertSANs = strings.RemoveDuplicate(append(k.APIServer.CertSANs, defaultKubeadmConfig.APIServer.CertSANs...))
	err = mergo.Merge(k, defaultKubeadmConfig)
	if err != nil {
		return fmt.Errorf("failed to merge kubeadm config: %v", err)
	}
	//using the DefaultKubeadmConfig configuration merge
	return k.Merge("")
}

func LoadKubeadmConfigs(arg string, decode func(arg string, kind string) (interface{}, error)) (*KubeadmConfig, error) {
	kubeadmConfig := &KubeadmConfig{}
	initConfig, err := decode(arg, InitConfiguration)
	if err != nil && err != io.EOF {
		return nil, err
	} else if initConfig != nil {
		kubeadmConfig.InitConfiguration = *initConfig.(*v1beta2.InitConfiguration)
	}
	clusterConfig, err := decode(arg, ClusterConfiguration)
	if err != nil && err != io.EOF {
		return nil, err
	} else if clusterConfig != nil {
		kubeadmConfig.ClusterConfiguration = *clusterConfig.(*v1beta2.ClusterConfiguration)
	}
	kubeProxyConfig, err := decode(arg, KubeProxyConfiguration)
	if err != nil && err != io.EOF {
		return nil, err
	} else if kubeProxyConfig != nil {
		kubeadmConfig.KubeProxyConfiguration = *kubeProxyConfig.(*v1alpha1.KubeProxyConfiguration)
	}
	kubeletConfig, err := decode(arg, KubeletConfiguration)
	if err != nil && err != io.EOF {
		return nil, err
	} else if kubeletConfig != nil {
		kubeadmConfig.KubeletConfiguration = *kubeletConfig.(*v1beta1.KubeletConfiguration)
	}
	joinConfig, err := decode(arg, JoinConfiguration)
	if err != nil && err != io.EOF {
		return nil, err
	} else if joinConfig != nil {
		kubeadmConfig.JoinConfiguration = *joinConfig.(*v1beta2.JoinConfiguration)
	}
	return kubeadmConfig, nil
}

func NewKubeadmConfig() interface{} {
	return &KubeadmConfig{}
}
