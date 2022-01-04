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
	"strings"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/logger"
)

var (
	ContainerdShell = `if grep "SystemdCgroup = true"  /etc/containerd/config.toml &> /dev/null; then  
driver=systemd
else
driver=cgroupfs
fi
echo ${driver}`
	DockerShell = `driver=$(docker info -f "{{.CgroupDriver}}")
	echo "${driver}"`
)

// k.getKubeVersion can't be empty
func (k *KubeadmRuntime) setKubeadmAPIVersion() {
	switch {
	case VersionCompare(k.getKubeVersion(), V1150) && !VersionCompare(k.getKubeVersion(), V1200):
		k.InitConfiguration.APIVersion = KubeadmV1beta2
		k.ClusterConfiguration.APIVersion = KubeadmV1beta2
		k.JoinConfiguration.APIVersion = KubeadmV1beta2
	// kubernetes gt 1.20, use Containerd instead of docker
	case VersionCompare(k.getKubeVersion(), V1200):
		k.InitConfiguration.APIVersion = KubeadmV1beta2
		k.ClusterConfiguration.APIVersion = KubeadmV1beta2
		k.JoinConfiguration.APIVersion = KubeadmV1beta2
	default:
		// Compatible with versions 1.14 and 1.13. but do not recommended.
		k.InitConfiguration.APIVersion = KubeadmV1beta1
		k.ClusterConfiguration.APIVersion = KubeadmV1beta1
		k.JoinConfiguration.APIVersion = KubeadmV1beta1
	}
}

// getCgroupDriverFromShell is get nodes container runtime CGroup by shell.
func (k *KubeadmRuntime) getCgroupDriverFromShell(node string) string {
	var cmd string
	if k.InitConfiguration.NodeRegistration.CRISocket == DefaultContainerdCRISocket {
		cmd = ContainerdShell
	} else {
		cmd = DockerShell
	}
	driver := k.CmdToString(node, cmd, " ")
	//driver, err := k.SSH.CmdToString(node, cmd, " ")
	if driver == "" {
		// by default if we get wrong output we set it default systemd?
		logger.Error("failed to get nodes [%s] cgroup driver", node)
		driver = DefaultSystemdCgroupDriver
	}
	driver = strings.TrimSpace(driver)
	logger.Debug("get nodes [%s] cgroup driver is [%s]", node, driver)
	return driver
}

func (k *KubeadmRuntime) MergeKubeadmConfig() error {
	if k.getKubeVersion() != "" {
		return nil
	}
	if k.Config.Clusterfile != "" && utils.IsFileExist(k.Config.Clusterfile) {
		if err := k.LoadFromClusterfile(k.Config.Clusterfile); err != nil {
			return fmt.Errorf("failed to load kubeadm config from clusterfile: %v", err)
		}
	}
	if err := k.Merge(k.getDefaultKubeadmConfig()); err != nil {
		return fmt.Errorf("failed to merge kubeadm config: %v", err)
	}
	k.setKubeadmAPIVersion()
	return nil
}
