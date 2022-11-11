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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	"github.com/sealerio/sealer/utils"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8snet "k8s.io/utils/net"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Vlog                         int
	VIP                          string
	RegistryInfo                 registry.Info
	containerRuntimeInfo         containerruntime.Info
	KubeadmConfigFromClusterFile kubeadm.KubeadmConfig
	APIServerDomain              string
}

// Runtime struct is the runtime interface for kubernetes
type Runtime struct {
	infra  infradriver.InfraDriver
	Config *Config
}

func NewKubeadmRuntime(clusterFileKubeConfig kubeadm.KubeadmConfig, infra infradriver.InfraDriver, containerRuntimeInfo containerruntime.Info, registryInfo registry.Info) (runtime.Installer, error) {
	k := &Runtime{
		infra: infra,
		Config: &Config{
			KubeadmConfigFromClusterFile: clusterFileKubeConfig,
			APIServerDomain:              DefaultAPIserverDomain,
			VIP:                          DefaultVIP,
			RegistryInfo:                 registryInfo,
			containerRuntimeInfo:         containerRuntimeInfo,
		},
	}

	if ipFamily := infra.GetClusterEnv()[common.EnvHostIPFamily]; ipFamily != nil && ipFamily.(string) == k8snet.IPv6 {
		k.Config.VIP = DefaultVIPForIPv6
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		k.Config.Vlog = 6
	}

	return k, nil
}

func (k *Runtime) Install() error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)
	workers := k.infra.GetHostIPListByRole(common.NODE)

	kubeadmConf, err := k.initKubeadmConfig(masters)
	if err != nil {
		return err
	}

	if err = k.generateCert(kubeadmConf, masters[0]); err != nil {
		return err
	}

	if err = k.createKubeConfig(masters[0]); err != nil {
		return err
	}

	if err = k.copyStaticFiles(masters[0:1]); err != nil {
		return err
	}

	token, certKey, err := k.initMaster0(kubeadmConf, masters[0])
	if err != nil {
		return err
	}

	if err = k.joinMasters(masters[1:], masters[0], kubeadmConf, token, certKey); err != nil {
		return err
	}

	if err = k.joinNodes(workers, masters, kubeadmConf, token); err != nil {
		return err
	}

	driver, err := k.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}

	if err := k.setRoles(driver); err != nil {
		return err
	}

	if err := k.dumpKubeConfigIntoCluster(driver, masters[0]); err != nil {
		return err
	}

	logrus.Info("Succeeded in creating a new cluster.")
	return nil
}

func (k *Runtime) GetCurrentRuntimeDriver() (runtime.Driver, error) {
	return NewKubeDriver(AdminKubeConfPath)
}

func (k *Runtime) Upgrade() error {
	panic("now not support upgrade")
}

func (k *Runtime) Reset() error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)
	workers := k.infra.GetHostIPListByRole(common.NODE)
	return k.reset(masters, workers)
}

func (k *Runtime) ScaleUp(newMasters, newWorkers []net.IP) error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)

	kubeadmConfig, err := kubeadm.LoadKubeadmConfigs(KubeadmFileYml, utils.DecodeCRDFromFile)
	if err != nil {
		return err
	}

	token, certKey, err := k.getJoinTokenHashAndKey(masters[0])
	if err != nil {
		return err
	}

	if err = k.joinMasters(newMasters, masters[0], kubeadmConfig, token, certKey); err != nil {
		return err
	}

	if err = k.joinNodes(newWorkers, masters, kubeadmConfig, token); err != nil {
		return err
	}

	driver, err := k.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}

	if err := k.setRoles(driver); err != nil {
		return err
	}

	logrus.Info("cluster scale up succeeded!")
	return nil
}

func (k *Runtime) ScaleDown(mastersToDelete, workersToDelete []net.IP) error {
	masters := k.infra.GetHostIPListByRole(common.MASTER)
	workers := k.infra.GetHostIPListByRole(common.NODE)

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
		remainWorkers := utilsnet.RemoveIPs(workers, workersToDelete)
		if err := k.deleteMasters(mastersToDelete, remainMasters, remainWorkers); err != nil {
			return err
		}
	}

	logrus.Info("cluster scale down succeeded!")
	return nil
}

// dumpKubeConfigIntoCluster save AdminKubeConf to cluster as secret resource.
func (k *Runtime) setRoles(driver runtime.Driver) error {
	nodeList := corev1.NodeList{}
	if err := driver.List(context.TODO(), &nodeList); err != nil {
		return err
	}

	for _, node := range nodeList.Items {
		addresses := node.Status.Addresses
		for _, address := range addresses {
			if address.Type != "InternalIP" {
				continue
			}
			roles := k.infra.GetRoleListByHostIP(address.Address)
			if len(roles) == 0 {
				continue
			}
			newNode := node.DeepCopy()
			for _, role := range roles {
				newNode.Labels["node-role.kubernetes.io/"+role] = ""
			}
			patch := runtimeClient.MergeFrom(&node)
			if err := driver.Patch(context.TODO(), newNode, patch); err != nil {
				return err
			}
		}
	}

	return nil
}

// dumpKubeConfigIntoCluster save AdminKubeConf to cluster as secret resource.
func (k *Runtime) dumpKubeConfigIntoCluster(driver runtime.Driver, master0 net.IP) error {
	kubeConfigContent, err := ioutil.ReadFile(AdminKubeConfPath)
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

// /var/lib/sealer/data/my-cluster/pki
func (k *Runtime) getPKIPath() string {
	return filepath.Join(k.infra.GetClusterRootfsPath(), "pki")
}

// /var/lib/sealer/data/my-cluster/pki/etcd
func (k *Runtime) getEtcdCertPath() string {
	return filepath.Join(k.getPKIPath(), "etcd")
}

// /var/lib/sealer/data/my-cluster/rootfs/statics
func (k *Runtime) getStaticFileDir() string {
	return filepath.Join(k.infra.GetClusterRootfsPath(), "statics")
}

// /var/lib/sealer/data/my-cluster/mount/etc/kubeadm.yml
func (k *Runtime) getDefaultKubeadmConfig() string {
	return filepath.Join(k.infra.GetClusterRootfsPath(), "etc", "kubeadm.yml")
}

func (k *Runtime) getAPIServerDomain() string {
	return k.Config.APIServerDomain
}

func (k *Runtime) getAPIServerVIP() net.IP {
	return net.ParseIP(k.Config.VIP)
}
