// Copyright © 2022 Alibaba Group Holding Ltd.
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
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config/v1beta2"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"
)

func decodeClusterFile(reader io.Reader, clusterfile *ClusterFile) error {
	decoder := yaml.NewYAMLToJSONDecoder(bufio.NewReaderSize(reader, 4096))

	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		if err := yaml.Unmarshal(ext.Raw, &metaType); err != nil {
			return fmt.Errorf("failed to decode TypeMeta: %v", err)
		}

		switch metaType.Kind {
		case common.Cluster:
			var cluster v2.Cluster

			if err := yaml.Unmarshal(ext.Raw, &cluster); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}
			clusterfile.cluster = &cluster

		case common.Config:
			var cfg v1.Config

			if err := yaml.Unmarshal(ext.Raw, &cfg); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			err := config.NewProcessorsAndRun(&cfg)
			if err != nil {
				return err
			}

			clusterfile.configs = append(clusterfile.configs, cfg)

		case common.Plugin:
			var plu v1.Plugin

			if err := yaml.Unmarshal(ext.Raw, &plu); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.plugins = append(clusterfile.plugins, plu)

		case common.InitConfiguration:
			var in v1beta2.InitConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.InitConfiguration = in

		case common.JoinConfiguration:
			var in v1beta2.JoinConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.JoinConfiguration = in

		case common.ClusterConfiguration:
			var in v1beta2.ClusterConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.ClusterConfiguration = in

		case common.KubeletConfiguration:
			var in v1beta1.KubeletConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.KubeletConfiguration = in

		case common.KubeProxyConfiguration:
			var in v1alpha1.KubeProxyConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.KubeProxyConfiguration = in
		}
	}
}
