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

package kubernetes

import (
	"fmt"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sealerio/sealer/pkg/runtime"
)

type kubeDriver struct {
	runtimeClient.Client
	kubeConfig string
}

func NewKubeDriver(kubeConfig string) (runtime.Driver, error) {
	client, err := GetClientFromConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to new k8s runtime client via adminconf: %v", err)
	}

	k := &kubeDriver{
		kubeConfig: kubeConfig,
	}

	k.Client = client
	return k, nil
}

// GetAdminKubeconfig returns the file path of admin kubeconfig is using.
func (k kubeDriver) GetAdminKubeconfig() string {
	return k.kubeConfig
}
