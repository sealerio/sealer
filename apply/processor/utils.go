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

package processor

import (
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/k0s"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func RuntimeChoose(rootfs string, cluster *v2.Cluster, config *kubeadm.KubeadmConfig) (runtime.Interface, error) {
	metadata, err := runtime.LoadMetadata(rootfs)
	if err != nil {
		return nil, err
	}
	switch metadata.ClusterRuntime {
	case runtime.K8s:
		return kubernetes.NewDefaultRuntime(cluster, config)
	case runtime.K0s:
		return k0s.NewK0sRuntime(cluster)
	// Todo case runtime.K3s:
	default:
		return kubernetes.NewDefaultRuntime(cluster, config)
	}
}
