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

package kubernetes

import (
	"fmt"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
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
	var kv kubeVersion
	kv = kv.Version(k.getKubeVersion())
	greatThanKV1150, err := kv.Compare(kv.Version(V1150))
	if err != nil {
		logrus.Errorf("compare kubernetes version failed: %s", err)
	}
	greatThanKV1230, err := kv.Compare(kv.Version(V1230))
	if err != nil {
		logrus.Errorf("compare kubernetes version failed: %s", err)
	}
	switch {
	case greatThanKV1150 && !greatThanKV1230:
		k.setAPIVersion(KubeadmV1beta2)
	case greatThanKV1230:
		k.setAPIVersion(KubeadmV1beta3)
	default:
		// Compatible with versions 1.14 and 1.13. but do not recommend.
		k.setAPIVersion(KubeadmV1beta1)
	}
}

// getCgroupDriverFromShell is get nodes container runtime CGroup by shell.
func (k *KubeadmRuntime) getCgroupDriverFromShell(node net.IP) (string, error) {
	var cmd string
	if k.InitConfiguration.NodeRegistration.CRISocket == DefaultContainerdCRISocket {
		cmd = ContainerdShell
	} else {
		cmd = DockerShell
	}
	driver, err := k.CmdToString(node, cmd, " ")
	if err != nil {
		return "", fmt.Errorf("failed to get nodes [%s] cgroup driver: %v", node, err)
	}
	if driver == "" {
		// by default if we get wrong output we set it default systemd?
		logrus.Errorf("failed to get nodes [%s] cgroup driver", node)
		driver = DefaultSystemdCgroupDriver
	}
	driver = strings.TrimSpace(driver)
	logrus.Debugf("get nodes [%s] cgroup driver is [%s]", node, driver)
	return driver, nil
}

func (k *KubeadmRuntime) MergeKubeadmConfig() error {
	if k.getKubeVersion() != "" {
		return nil
	}
	if k.Config.ClusterFileKubeConfig != nil {
		if err := k.LoadFromClusterfile(k.Config.ClusterFileKubeConfig); err != nil {
			return fmt.Errorf("failed to load kubeadm config from clusterfile: %v", err)
		}
	}
	if err := k.Merge(k.getDefaultKubeadmConfig()); err != nil {
		return fmt.Errorf("failed to merge kubeadm config: %v", err)
	}
	k.setKubeadmAPIVersion()
	return nil
}
