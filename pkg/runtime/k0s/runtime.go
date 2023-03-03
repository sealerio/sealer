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

package k0s

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"

	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	utilsnet "github.com/sealerio/sealer/utils/net"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Runtime struct is the runtime interface for k0s
type Runtime struct {
	infra                infradriver.InfraDriver
	Vlog                 int
	containerRuntimeInfo containerruntime.Info
	registryInfo         registry.Info
	//TODO：now do not support custom APIServerDomain
}

// NewK0sRuntime gen a k0s bootstrap process that leading k0s cluster management
func NewK0sRuntime(infra infradriver.InfraDriver, containerRuntimeInfo containerruntime.Info, registryInfo registry.Info) (runtime.Installer, error) {
	k := &Runtime{
		infra:                infra,
		registryInfo:         registryInfo,
		containerRuntimeInfo: containerRuntimeInfo,
	}

	setDebugLevel(k)
	return k, nil
}

func (k *Runtime) Install() error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)
	workers := k.infra.GetHostIPListByRole(common.NODE)

	if err := k.initKube([]net.IP{masters[0]}); err != nil {
		return err
	}
	// registryInfo like "sea.hub:5000" is needed which used to modify the k0s.yaml private registry repo
	if err := k.generateConfigOnMaster0(masters[0], k.registryInfo.URL); err != nil {
		return err
	}

	if err := k.bootstrapMaster0(masters[0]); err != nil {
		return err
	}

	if err := k.generateJoinToken(masters[0]); err != nil {
		return err
	}

	if err := k.joinMasters(masters[1:], k.registryInfo.URL); err != nil {
		return err
	}

	if err := k.joinNodes(workers); err != nil {
		return err
	}

	driver, err := k.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}

	if err := k.dumpKubeConfigIntoCluster(driver, masters[0]); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) GetCurrentRuntimeDriver() (runtime.Driver, error) {
	return NewKubeDriver(DefaultAdminConfPath)
}

func (k *Runtime) ScaleUp(newMasters, newWorkers []net.IP) error {
	if err := k.joinMasters(newMasters, k.registryInfo.URL); err != nil {
		return err
	}

	if err := k.joinNodes(newWorkers); err != nil {
		return err
	}
	logrus.Info("cluster scale up succeeded!")
	return nil
}

func (k *Runtime) ScaleDown(mastersToDelete, workersToDelete []net.IP) error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)

	remainMasters := utilsnet.RemoveIPs(masters, mastersToDelete)
	if len(remainMasters) == 0 {
		return fmt.Errorf("cleaning up all masters is illegal, unless you give the --all flag, which will delete the entire cluster")
	}

	if len(workersToDelete) > 0 {
		if err := k.deleteNodes(workersToDelete, remainMasters); err != nil {
			return err
		}
	}

	if len(mastersToDelete) > 0 {
		if err := k.deleteMasters(mastersToDelete, remainMasters); err != nil {
			return err
		}
	}
	logrus.Info("cluster scale down succeeded!")
	return nil
}

func setDebugLevel(k *Runtime) {
	if logrus.GetLevel() == logrus.DebugLevel {
		k.Vlog = 6
	}
}

func (k *Runtime) Upgrade() error {
	panic("implement me")
}

func (k *Runtime) Reset() error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)
	workers := k.infra.GetHostIPListByRole(common.NODE)
	return k.reset(masters, workers)
}

func (k *Runtime) CopyJoinToken(role string, hosts []net.IP) error {
	var joinCertPath string
	switch role {
	case ControllerRole:
		joinCertPath = DefaultK0sControllerJoin
	case WorkerRole:
		joinCertPath = DefaultK0sWorkerJoin
	default:
		joinCertPath = DefaultK0sWorkerJoin
	}

	eg, _ := errgroup.WithContext(context.Background())
	for _, host := range hosts {
		host := host
		eg.Go(func() error {
			return k.infra.Copy(host, joinCertPath, joinCertPath)
		})
	}
	return nil
}

func (k *Runtime) JoinCommand(role, registryInfo string) []string {
	cmds := map[string][]string{
		ControllerRole: {"mkdir -p /etc/k0s", fmt.Sprintf("k0s config create > %s", DefaultK0sConfigPath),
			fmt.Sprintf("sed -i '/  images/ a\\    repository: \"%s\"' %s", registryInfo, DefaultK0sConfigPath),
			fmt.Sprintf("k0s install controller --token-file %s -c %s --cri-socket %s",
				DefaultK0sControllerJoin, DefaultK0sConfigPath, ExternalCRIAddress),
			"k0s start",
		},
		WorkerRole: {fmt.Sprintf("k0s install worker --cri-socket %s --token-file %s", ExternalCRIAddress, DefaultK0sWorkerJoin),
			"k0s start"},
	}

	v, ok := cmds[role]
	if !ok {
		return nil
	}
	return v
}

// dumpKubeConfigIntoCluster save AdminKubeConf to cluster as secret resource.
func (k *Runtime) dumpKubeConfigIntoCluster(driver runtime.Driver, master0 net.IP) error {
	kubeConfigContent, err := os.ReadFile(DefaultAdminConfPath)
	if err != nil {
		return err
	}

	kubeConfigContent = bytes.ReplaceAll(kubeConfigContent, []byte("apiserver.cluster.local"), []byte(master0.String()))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin.conf",
			Namespace: metav1.NamespaceSystem,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"admin.conf": kubeConfigContent,
		},
	}

	if err := driver.Create(context.Background(), secret, &runtimeClient.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create secret: %v", err)
		}

		if err := driver.Update(context.Background(), secret, &runtimeClient.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update secret: %v", err)
		}
	}

	return nil
}
