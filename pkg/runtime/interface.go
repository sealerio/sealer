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

package runtime

import (
	"net"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installer interface {
	// Install exec init phase for cluster. TODO: make the annotation more comprehensive
	Install() error

	GetCurrentRuntimeDriver() (Driver, error)

	// Reset exec reset phase for cluster.TODO: make the annotation more comprehensive
	Reset() error
	// ScaleUp exec joining phase for cluster, add master role for these nodes. net.IP is the master node IP array.
	ScaleUp(newMasters, newWorkers []net.IP) error
	// ScaleDown exec deleting phase for deleting cluster master role nodes. net.IP is the master node IP array.
	ScaleDown(mastersToDelete, workersToDelete []net.IP) error

	// Upgrade exec upgrading phase for cluster.TODO: make the annotation more comprehensive
	Upgrade() error
}

type Driver interface {
	client.Client

	GetAdminKubeconfig() string
}
