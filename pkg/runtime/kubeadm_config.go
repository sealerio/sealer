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

package runtime

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/runtime/kubeadm_types/v1beta2"
	"github.com/alibaba/sealer/utils"
	"github.com/imdario/mergo"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"
)

// Read config from https://github.com/alibaba/sealer/blob/main/docs/design/clusterfile-v2.md and overwrite default kubeadm.yaml
// Use github.com/imdario/mergo to merge kubeadm config in Clusterfile and the default kubeadm config
// Using a config filter to handle some edge cases

// https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/apis/kubeadm/v1beta2/types.go
// Using map to overwrite Kubeadm configs

//nolint
type KubeadmConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	KubeConfigSpec    `json:"spec,omitempty"`
}

//nolint
type KubeConfigSpec struct {
	v1beta2.InitConfiguration
	v1beta2.ClusterConfiguration
	v1alpha1.KubeProxyConfiguration
	v1beta1.KubeletConfiguration
	v1beta2.JoinConfiguration
}

// Load KubeadmConfig from Clusterfile
// If has `KubeadmConfig` in Clusterfile, load every field to each configurations
// If Kubeadm raw config in Clusterfile, just load it
func (k *KubeadmConfig) LoadFromClusterfile(fileName string) error {
	kubeConfig, err := DecodeCRDFromFile(fileName, Kubeadmconfig)
	if err != nil {
		return err
	} else if kubeConfig != nil {
		k.KubeConfigSpec = kubeConfig.(*KubeadmConfig).KubeConfigSpec
	}

	kubeadmConfig, err := k.loadKubeadmConfigs(fileName, DecodeCRDFromFile)
	if err != nil {
		return fmt.Errorf("failed to load kubeadm config from %s, err: %v", fileName, err)
	} else if kubeConfig == nil {
		return nil
	}
	return mergo.Merge(k, kubeadmConfig)
}

// Merge Using github.com/imdario/mergo to merge KubeadmConfig to the CloudImage default kubeadm config, overwrite some field.
// if defaultKubeadmConfig file not exist, use default raw kubeadm config to merge k.KubeConfigSpec empty value
func (k *KubeadmConfig) Merge(kubeadmYamlPath string) ([]byte, error) {
	var (
		defaultKubeadmConfig *KubeadmConfig
		err                  error
	)
	if utils.IsFileExist(kubeadmYamlPath) {
		defaultKubeadmConfig, err = k.loadKubeadmConfigs(kubeadmYamlPath, DecodeCRDFromFile)
		if err != nil {
			logger.Warn("failed to found kubeadm config from %s : %v, will use default kubeadm config to merge empty value", kubeadmYamlPath, err)
			return k.Merge("")
		}
	} else {
		defaultKubeadmConfig, err = k.loadKubeadmConfigs(DefaultKubeadmConfig, DecodeCRDFromString)
		if err != nil {
			return nil, err
		}
	}

	if err = mergo.Merge(k, defaultKubeadmConfig); err != nil {
		return nil, err
	}
	return utils.MarshalConfigsToYaml(&k.InitConfiguration,
		&k.ClusterConfiguration,
		&k.KubeletConfiguration,
		&k.KubeProxyConfiguration)
}

func (k *KubeadmConfig) loadKubeadmConfigs(arg string, decode func(arg string, kind string) (interface{}, error)) (*KubeadmConfig, error) {
	kubeadmConfig := &KubeadmConfig{}
	initConfig, err := decode(arg, InitConfiguration)
	if err != nil {
		return nil, err
	} else if initConfig != nil {
		kubeadmConfig.InitConfiguration = *initConfig.(*v1beta2.InitConfiguration)
	}
	clusterConfig, err := decode(arg, ClusterConfiguration)
	if err != nil {
		return nil, err
	} else if clusterConfig != nil {
		kubeadmConfig.ClusterConfiguration = *clusterConfig.(*v1beta2.ClusterConfiguration)
	}
	kubeProxyConfig, err := decode(arg, KubeProxyConfiguration)
	if err != nil {
		return nil, err
	} else if kubeProxyConfig != nil {
		kubeadmConfig.KubeProxyConfiguration = *kubeProxyConfig.(*v1alpha1.KubeProxyConfiguration)
	}
	kubeletConfig, err := decode(arg, KubeletConfiguration)
	if err != nil {
		return nil, err
	} else if kubeletConfig != nil {
		kubeadmConfig.KubeletConfiguration = *kubeletConfig.(*v1beta1.KubeletConfiguration)
	}
	joinConfig, err := decode(arg, JoinConfiguration)
	if err != nil {
		return nil, err
	} else if joinConfig != nil {
		kubeadmConfig.JoinConfiguration = *joinConfig.(*v1beta2.JoinConfiguration)
	}
	return kubeadmConfig, nil
}

func NewKubeadmConfig() interface{} {
	return &KubeadmConfig{}
}
