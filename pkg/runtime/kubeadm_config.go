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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/runtime/kubeadm_types/v1beta2"
	"github.com/alibaba/sealer/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"
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
type KubeadmConfig struct {
	*v1beta2.InitConfiguration
	*v1beta2.ClusterConfiguration
	*v1alpha1.KubeProxyConfiguration
	*v1beta1.KubeletConfiguration
}

// Load KubeadmConfig from Clusterfile
// If has `KubeadmConfig` in Clusterfile, load every field to each configurations
// If Kubeadm raw config in Clusterfile, just load it
func (k *KubeadmConfig) LoadFromClusterfile(fileName string) error {
	if err := k.init(fileName); err != nil {
		return err
	}
	kubeConfig, err := utils.DecodeCRDFromFile(fileName, common.KubeConfig)
	if err != nil {
		return err
	}
	// type conversion
	customConfig := KubeadmConfig{
		InitConfiguration:      &kubeConfig.(*v2.KubeConfig).Spec.InitConfiguration,
		ClusterConfiguration:   &kubeConfig.(*v2.KubeConfig).Spec.ClusterConfiguration,
		KubeletConfiguration:   &kubeConfig.(*v2.KubeConfig).Spec.KubeletConfiguration,
		KubeProxyConfiguration: &kubeConfig.(*v2.KubeConfig).Spec.KubeProxyConfiguration,
	}
	return mergo.Merge(k, customConfig, mergo.WithOverride, mergo.WithOverwriteWithEmptyValue)
}

// Using github.com/imdario/mergo to merge KubeadmConfig to the CloudImage default kubeadm config, overwrite some field.
func (k *KubeadmConfig) Merge(defaultKubeadmConfig string) ([]byte, error) {
	/**
	1. use rootfs/etc/kubeadm.yaml, if kubeadm.yaml not exist, use default raw kubeadm config
	2. kubeadm config from Clusterfile > ⬆ merge  WithOverride
	3. kubeConfig from Clusterfile > ⬆ merge WithOverride
	**/

	/* defaultKubeadmConfig = runtime.GetInitKubeadmConfigYaml("my-cluster") */

	defaultKubeConfig, err := utils.DecodeCRDFromString(defaultKubeadmConfig, common.KubeConfig)
	if err != nil {
		return nil, err
	}
	if defaultKubeConfig == nil {
		logger.Warn("kubeadm config not found in cloud image")
		defaultKubeadmConfig = runtime.GetInitKubeadmConfigYaml("")
		return k.Merge(defaultKubeadmConfig)
	}
	defaultConfig := KubeadmConfig{
		InitConfiguration:      &defaultKubeConfig.(*v2.KubeConfig).Spec.InitConfiguration,
		ClusterConfiguration:   &defaultKubeConfig.(*v2.KubeConfig).Spec.ClusterConfiguration,
		KubeletConfiguration:   &defaultKubeConfig.(*v2.KubeConfig).Spec.KubeletConfiguration,
		KubeProxyConfiguration: &defaultKubeConfig.(*v2.KubeConfig).Spec.KubeProxyConfiguration,
	}
	/*	err = k.LoadFromClusterfile("clusterfile")
		if err != nil {
			return nil, err
		}*/

	//Override default kubeadm config
	err = mergo.Merge(&defaultConfig, *k, mergo.WithOverride)
	if err != nil {
		return nil, fmt.Errorf("failed to merge default kube config, err: %v", err)
	}
	return utils.MarshalConfigYaml(&defaultConfig.InitConfiguration,
		&defaultConfig.ClusterConfiguration,
		&defaultConfig.KubeletConfiguration,
		&defaultConfig.KubeProxyConfiguration)
}

func (k *KubeadmConfig) init(fileName string) error {
	initConfig, err := utils.DecodeCRDFromFile(fileName, common.InitConfiguration)
	if err != nil {
		return err
	}
	clusterConfig, err := utils.DecodeCRDFromFile(fileName, common.ClusterConfiguration)
	if err != nil {
		return err
	}
	kubeProxyConfig, err := utils.DecodeCRDFromFile(fileName, common.KubeProxyConfiguration)
	if err != nil {
		return err
	}
	kubeletConfig, err := utils.DecodeCRDFromFile(fileName, common.KubeletConfiguration)
	if err != nil {
		return err
	}
	k.InitConfiguration = initConfig.(*v1beta2.InitConfiguration)
	k.ClusterConfiguration = clusterConfig.(*v1beta2.ClusterConfiguration)
	k.KubeletConfiguration = kubeletConfig.(*v1beta1.KubeletConfiguration)
	k.KubeProxyConfiguration = kubeProxyConfig.(*v1alpha1.KubeProxyConfiguration)
	return nil
}

func NewKubeadmConfig() interface{} {
	return &KubeadmConfig{}
}
