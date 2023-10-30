// Copyright © 2023 Alibaba Group Holding Ltd.
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

package preflight

import (
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/client/k8s"
)

// Checker validates the desired value
type Checker interface {
	Check() (warnings, errorList []error)
	Name() string
}

// ClusterIsExistsCheck verifies the given cluster is existed.
type ClusterIsExistsCheck struct {
}

// Name returns label for checker.
func (ClusterIsExistsCheck) Name() string {
	return "ClusterExist"
}

// Check :if the given cluster is existed, return error.
func (c ClusterIsExistsCheck) Check() (warnings, errorList []error) {
	// check K8s client
	client, _ := k8s.NewK8sClient()
	if client != nil {
		return nil, []error{errors.Errorf("cluster is exist: could new k8s client via default kubeconfig")}
	}

	return nil, nil
}
