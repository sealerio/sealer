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
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"

	"github.com/alibaba/sealer/pkg/runtime/kubeadm_types/v1beta2"
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
	// TODO
	return nil
}

// Using github.com/imdario/mergo to merge KubeadmConfig to the CloudImage default kubeadm config, overwrite some field.
func (k *KubeadmConfig) Merge(defaultKubeadmConfig string) ([]byte, error) {
	// TODO
	return []byte{}, nil
}
