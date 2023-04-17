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

package clusterruntime

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/k0s"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var tryTimes = 10
var trySleepTime = time.Second

const (
	// CRILabel is key for get container runtime interface type.
	CRILabel = "cluster.alpha.sealer.io/container-runtime-type"
	// CRTLabel is key for get cluster runtime type.
	CRTLabel = "cluster.alpha.sealer.io/cluster-runtime-type"

	RegistryConfigMapName     = "sealer-registry"
	RegistryConfigMapDataName = "registry"
)

// RuntimeConfig for Installer
type RuntimeConfig struct {
	Distributor            imagedistributor.Distributor
	ContainerRuntimeConfig v2.ContainerRuntimeConfig
	KubeadmConfig          kubeadm.KubeadmConfig
	Plugins                []v1.Plugin
}

type Installer struct {
	RuntimeConfig
	infraDriver               infradriver.InfraDriver
	containerRuntimeInstaller containerruntime.Installer
	clusterRuntimeType        string
	hooks                     map[Phase]HookConfigList
	regConfig                 v2.Registry
}

type InstallInfo struct {
	ContainerRuntimeType string
	ClusterRuntimeType   string
}

func getCRIInstaller(containerRuntime string, infraDriver infradriver.InfraDriver) (containerruntime.Installer, error) {
	switch containerRuntime {
	case common.Docker:
		return containerruntime.NewInstaller(v2.ContainerRuntimeConfig{
			Type: common.Docker,
		}, infraDriver)
	case common.Containerd:
		return containerruntime.NewInstaller(v2.ContainerRuntimeConfig{
			Type: common.Containerd,
		}, infraDriver)
	default:
		return nil, fmt.Errorf("not support container runtime %s", containerRuntime)
	}
}

func getClusterRuntimeInstaller(clusterRuntimeType string, driver infradriver.InfraDriver, crInfo containerruntime.Info,
	registryInfo registry.Info, kubeadmConfig kubeadm.KubeadmConfig) (runtime.Installer, error) {
	switch clusterRuntimeType {
	case common.K8s:
		return kubernetes.NewKubeadmRuntime(kubeadmConfig, driver, crInfo, registryInfo)
	case common.K0s:
		return k0s.NewK0sRuntime(driver, crInfo, registryInfo)
		//todo support k3s runtime
	default:
		return nil, fmt.Errorf("not support cluster runtime %s", clusterRuntimeType)
	}
}

func NewInstaller(infraDriver infradriver.InfraDriver, runtimeConfig RuntimeConfig, installInfo InstallInfo) (*Installer, error) {
	var (
		err       error
		installer = &Installer{
			regConfig:          infraDriver.GetClusterRegistry(),
			clusterRuntimeType: installInfo.ClusterRuntimeType,
		}
	)

	installer.RuntimeConfig = runtimeConfig
	// configure container runtime
	//todo need to support other container runtimes
	installer.containerRuntimeInstaller, err = getCRIInstaller(installInfo.ContainerRuntimeType, infraDriver)
	if err != nil {
		return nil, err
	}

	// add installer hooks
	hooks, err := transferPluginsToHooks(runtimeConfig.Plugins)
	if err != nil {
		return nil, err
	}
	installer.hooks = hooks
	installer.infraDriver = infraDriver
	return installer, nil
}

func (i *Installer) Install() error {
	var (
		masters = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0 = masters[0]
		workers = getWorkerIPList(i.infraDriver)
		all     = append(masters, workers...)
		rootfs  = i.infraDriver.GetClusterRootfsPath()
	)

	// set HostAlias
	if err := i.infraDriver.SetClusterHostAliases(all); err != nil {
		return err
	}

	// distribute rootfs
	if err := i.Distributor.Distribute(all, rootfs); err != nil {
		return err
	}

	if err := i.runClusterHook(master0, PreInstallCluster); err != nil {
		return err
	}

	if err := i.runHostHook(PreInitHost, all); err != nil {
		return err
	}

	if err := i.containerRuntimeInstaller.InstallOn(all); err != nil {
		return err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return err
	}

	var deployHosts []net.IP
	if i.regConfig.LocalRegistry != nil {
		installer := registry.NewInstaller(nil, i.regConfig.LocalRegistry, i.infraDriver, i.Distributor)
		if *i.regConfig.LocalRegistry.HA {
			deployHosts, err = installer.Reconcile(masters)
			if err != nil {
				return err
			}
		} else {
			deployHosts, err = installer.Reconcile([]net.IP{master0})
			if err != nil {
				return err
			}
		}
	}

	registryConfigurator, err := registry.NewConfigurator(deployHosts, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	if err = registryConfigurator.InstallOn(masters, workers); err != nil {
		return err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return err
	}

	registryInfo := registryConfigurator.GetRegistryInfo()

	kubeRuntimeInstaller, err := getClusterRuntimeInstaller(i.clusterRuntimeType, i.infraDriver,
		crInfo, registryDriver.GetInfo(), i.KubeadmConfig)
	if err != nil {
		return err
	}

	if err = kubeRuntimeInstaller.Install(); err != nil {
		return err
	}

	if err = i.runHostHook(PostInitHost, all); err != nil {
		return err
	}

	if err = i.runClusterHook(master0, PostInstallCluster); err != nil {
		return err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}

	if err := i.setRoles(runtimeDriver); err != nil {
		return err
	}

	if err = i.setNodeLabels(all, runtimeDriver); err != nil {
		return err
	}

	if err = i.setNodeTaints(all, runtimeDriver); err != nil {
		return err
	}

	if err = i.saveRegistryInfo(runtimeDriver, registryInfo); err != nil {
		return err
	}

	return nil
}

func (i *Installer) GetCurrentDriver() (registry.Driver, runtime.Driver, error) {
	var (
		masters             = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0             = masters[0]
		registryDeployHosts = masters
	)
	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	if i.regConfig.LocalRegistry != nil && !*i.regConfig.LocalRegistry.HA {
		registryDeployHosts = []net.IP{master0}
	}
	// TODO, init here or in constructor?
	registryConfigurator, err := registry.NewConfigurator(registryDeployHosts, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
		return nil, nil, err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return nil, nil, err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, crInfo, registryDriver.GetInfo())
	if err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

// setRoles save roles
func (i *Installer) setRoles(driver runtime.Driver) error {
	nodeList := corev1.NodeList{}
	if err := driver.List(context.TODO(), &nodeList); err != nil {
		return err
	}

	genRoleLabelFunc := func(role string) string {
		return fmt.Sprintf("node-role.kubernetes.io/%s", role)
	}

	for idx, node := range nodeList.Items {
		addresses := node.Status.Addresses
		for _, address := range addresses {
			if address.Type != "InternalIP" {
				continue
			}
			roles := i.infraDriver.GetRoleListByHostIP(address.Address)
			if len(roles) == 0 {
				continue
			}
			newNode := node.DeepCopy()

			for _, role := range roles {
				newNode.Labels[genRoleLabelFunc(role)] = ""
			}
			patch := runtimeClient.MergeFrom(&nodeList.Items[idx])

			if err := driver.Patch(context.TODO(), newNode, patch); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Installer) setNodeLabels(hosts []net.IP, driver runtime.Driver) error {
	// set new added host labels if it is existed
	nodeList := corev1.NodeList{}
	if err := driver.List(context.TODO(), &nodeList); err != nil {
		return fmt.Errorf("failed to list cluster nodes: %v", err)
	}

	nodeLabel := make(map[string]corev1.Node)
	for _, node := range nodeList.Items {
		nodeLabel[getAddress(node.Status.Addresses)] = node
	}

	for _, ip := range hosts {
		labels := i.infraDriver.GetHostLabels(ip)
		if len(labels) == 0 {
			continue
		}

		if node, ok := nodeLabel[ip.String()]; ok {
			newNode := node.DeepCopy()
			m := node.GetLabels()
			for key, value := range labels {
				m[key] = value
			}

			newNode.SetLabels(m)
			newNode.SetResourceVersion("")
			if err := driver.Update(context.TODO(), newNode); err != nil {
				return fmt.Errorf("failed to label cluster nodes %s: %v", ip.String(), err)
			}
		}
	}

	return nil
}

func getAddress(addresses []corev1.NodeAddress) string {
	for _, v := range addresses {
		if strings.EqualFold(string(v.Type), "InternalIP") {
			return v.Address
		}
	}
	return ""
}

func (i *Installer) setNodeTaints(hosts []net.IP, driver runtime.Driver) error {
	var (
		k8snode    corev1.Node
		ok         bool
		nodeTaints []corev1.Taint
	)
	nodeList := corev1.NodeList{}
	if err := driver.List(context.TODO(), &nodeList); err != nil {
		return fmt.Errorf("failed to list cluster nodes: %v", err)
	}
	nodeTaint := make(map[string]corev1.Node)
	for _, node := range nodeList.Items {
		nodeTaint[getAddress(node.Status.Addresses)] = node
	}

	for _, ip := range hosts {
		taints := i.infraDriver.GetHostTaints(ip)
		if len(taints) == 0 {
			continue
		}

		if k8snode, ok = nodeTaint[ip.String()]; !ok {
			continue
		}
		newNode := k8snode.DeepCopy()
		for _, taint := range taints {
			if strings.Contains(taint.Key, infradriver.DelSymbol) {
				taintKey := strings.TrimSuffix(taint.Key, infradriver.DelSymbol)
				nodeTaints, _ = infradriver.DeleteTaintsByKey(newNode.Spec.Taints, taintKey)
				newNode.Spec.Taints = nodeTaints
			} else if strings.Contains(string(taint.Effect), infradriver.DelSymbol) {
				nodeTaints, _ = infradriver.DeleteTaint(newNode.Spec.Taints, &taint) // #nosec
				newNode.Spec.Taints = nodeTaints
			} else {
				newNode.Spec.Taints = taints
			}
		}
		newNode.SetResourceVersion("")
		if err := utils.Retry(tryTimes, trySleepTime, func() error {
			if err := driver.Update(context.TODO(), newNode); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) saveRegistryInfo(driver runtime.Driver, registryInfo registry.RegistryInfo) error {
	info, err := yaml.Marshal(registryInfo)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RegistryConfigMapName,
			Namespace: clusterfile.ClusterfileConfigMapNamespace,
		},
		Data: map[string]string{RegistryConfigMapDataName: string(info)},
	}

	ctx := context.Background()
	if err := driver.Create(ctx, cm); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configmap: %v", err)
		}

		if err := driver.Update(ctx, cm); err != nil {
			return fmt.Errorf("unable to update configmap: %v", err)
		}
	}
	return nil
}

func GetClusterInstallInfo(imageLabels map[string]string, criConfig v2.ContainerRuntimeConfig) InstallInfo {
	cri := imageLabels[CRILabel]
	if cri == "" {
		cri = common.Docker
	}
	if criConfig.Type != "" {
		cri = criConfig.Type
	}
	clusterRuntimeType := imageLabels[CRTLabel]
	if clusterRuntimeType == "" {
		clusterRuntimeType = common.K8s
	}
	logrus.Infof("The cri is %s, cluster runtime type is %s\n", cri, clusterRuntimeType)
	return InstallInfo{
		ContainerRuntimeType: cri,
		ClusterRuntimeType:   clusterRuntimeType,
	}
}

func GetClusterConfPath(labels map[string]string) string {
	clusterRuntimeType := labels[CRTLabel]
	if clusterRuntimeType == "" {
		clusterRuntimeType = common.K8s
	}
	switch clusterRuntimeType {
	case common.K8s:
		return kubernetes.AdminKubeConfPath
	case common.K0s:
		return k0s.DefaultAdminConfPath
	//TODO support k3s
	default:
		return kubernetes.AdminKubeConfPath
	}
}
