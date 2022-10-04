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
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
	strUtils "github.com/sealerio/sealer/utils/strings"
	versionUtils "github.com/sealerio/sealer/utils/version"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"
)

type Config struct {
	// Clusterfile: the absolute path, we need to read kubeadm config from Clusterfile
	ClusterFileKubeConfig *kubeadm.KubeadmConfig
}

// Runtime struct is the runtime interface for kubernetes
type Runtime struct {
	*sync.Mutex
	// TODO: remove field cluster from runtime pkg
	// just make runtime to use essential data from cluster, rather than the whole cluster scope.
	cluster *v2.Cluster

	// The KubeadmConfig used to setup the final cluster.
	// Its data is from KubeadmConfig input from Clusterfile and KubeadmConfig in ClusterImage.
	*kubeadm.KubeadmConfig
	// *Config
	KubeadmConfigFromClusterfile *kubeadm.KubeadmConfig
	APIServerDomain              string
	Vlog                         int
	VIP                          string

	// RegConfig contains the embedded registry configuration of cluster
	RegConfig *registry.Config
	*k8s.Client
}

// NewDefaultRuntime arg "clusterfileKubeConfig" is the Clusterfile path/name, runtime need read kubeadm config from it
// Mount image is required before new Runtime.
func NewDefaultRuntime(cluster *v2.Cluster, clusterfileKubeConfig *kubeadm.KubeadmConfig) (runtime.Interface, error) {
	return newKubernetesRuntime(cluster, clusterfileKubeConfig)
}

func newKubernetesRuntime(cluster *v2.Cluster, clusterFileKubeConfig *kubeadm.KubeadmConfig) (runtime.Interface, error) {
	k := &Runtime{
		cluster: cluster,
		/*Config: &Config{
			ClusterFileKubeConfig: clusterFileKubeConfig,
		},*/
		KubeadmConfigFromClusterfile: clusterFileKubeConfig,
		KubeadmConfig:                &kubeadm.KubeadmConfig{},
		APIServerDomain:              DefaultAPIserverDomain,
	}

	var err error
	if k.Client, err = k8s.Newk8sClient(); err != nil {
		// In current design, as runtime controls all cluster operations including run, join, delete
		// and so on, then when executing run operation, it will definitely fail when creating k8s client
		// since no k8s cluster is setup. While when join and delete operation, the cluster already exists,
		// we can make it to create k8s client. Therefore just throw a warn log to move on.
		logrus.Warnf("failed to create k8s client: %v", err)
	}

	k.RegConfig = registry.GetConfig(k.getImageMountDir(), k.cluster.GetMaster0IP())
	k.setCertSANS(append(
		[]string{"127.0.0.1", k.getAPIServerDomain(), k.getVIP().String()},
		k.cluster.GetMasterIPStrList()...),
	)
	// TODO args pre checks
	if err := k.checkList(); err != nil {
		return nil, err
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		k.Vlog = 6
	}
	return k, nil
}

func (k *Runtime) Init() error {
	return k.init()
}

func (k *Runtime) Upgrade() error {
	return k.upgrade()
}

func (k *Runtime) Reset() error {
	logrus.Infof("Start to delete cluster: master %s, node %s", k.cluster.GetMasterIPList(), k.cluster.GetNodeIPList())
	if err := k.confirmDeleteNodes(); err != nil {
		return err
	}
	return k.reset()
}

func (k *Runtime) JoinMasters(newMastersIPList []net.IP) error {
	if len(newMastersIPList) != 0 {
		logrus.Infof("%s will be added as master", newMastersIPList)
	}
	return k.joinMasters(newMastersIPList)
}

func (k *Runtime) JoinNodes(newNodesIPList []net.IP) error {
	if len(newNodesIPList) != 0 {
		logrus.Infof("%s will be added as worker", newNodesIPList)
	}
	return k.joinNodes(newNodesIPList)
}

func (k *Runtime) DeleteMasters(mastersIPList []net.IP) error {
	if len(mastersIPList) != 0 {
		logrus.Infof("master %s will be deleted", mastersIPList)
		if err := k.confirmDeleteNodes(); err != nil {
			return err
		}
	}
	return k.deleteMasters(mastersIPList)
}

func (k *Runtime) DeleteNodes(nodesIPList []net.IP) error {
	if len(nodesIPList) != 0 {
		logrus.Infof("worker %s will be deleted", nodesIPList)
		if err := k.confirmDeleteNodes(); err != nil {
			return err
		}
	}
	return k.deleteNodes(nodesIPList)
}

func (k *Runtime) confirmDeleteNodes() error {
	if !runtime.ForceDelete {
		if pass, err := utils.ConfirmOperation("Are you sure to delete these nodes? "); err != nil {
			return err
		} else if !pass {
			return fmt.Errorf("exit the operation of delete these nodes")
		}
	}
	return nil
}

func (k *Runtime) GetClusterMetadata() (*runtime.Metadata, error) {
	return k.getClusterMetadata()
}

func (k *Runtime) checkList() error {
	if len(k.cluster.Spec.Hosts) == 0 {
		return fmt.Errorf("master hosts cannot be empty")
	}
	if k.cluster.GetMaster0IP() == nil {
		return fmt.Errorf("master hosts ip cannot be empty")
	}
	return nil
}

func (k *Runtime) getClusterMetadata() (*runtime.Metadata, error) {
	metadata := &runtime.Metadata{}
	if k.getKubeVersion() == "" {
		if err := k.MergeKubeadmConfig(); err != nil {
			return nil, err
		}
	}
	metadata.Version = k.getKubeVersion()
	return metadata, nil
}

func (k *Runtime) getHostSSHClient(hostIP net.IP) (ssh.Interface, error) {
	return ssh.NewStdoutSSHClient(hostIP, k.cluster)
}

// /var/lib/sealer/data/my-cluster
func (k *Runtime) getBasePath() string {
	return common.DefaultClusterBaseDir(k.cluster.Name)
}

// /var/lib/sealer/data/my-cluster/rootfs
func (k *Runtime) getRootfs() string {
	return common.DefaultTheClusterRootfsDir(k.cluster.Name)
}

// /var/lib/sealer/data/my-cluster/mount
func (k *Runtime) getImageMountDir() string {
	return platform.DefaultMountClusterImageDir(k.cluster.Name)
}

// /var/lib/sealer/data/my-cluster/certs
func (k *Runtime) getCertsDir() string {
	return common.TheDefaultClusterCertDir(k.cluster.Name)
}

// /var/lib/sealer/data/my-cluster/pki
func (k *Runtime) getPKIPath() string {
	return common.TheDefaultClusterPKIDir(k.cluster.Name)
}

// /var/lib/sealer/data/my-cluster/mount/etc/kubeadm.yml
func (k *Runtime) getDefaultKubeadmConfig() string {
	return filepath.Join(k.getImageMountDir(), "etc", "kubeadm.yml")
}

// /var/lib/sealer/data/my-cluster/pki/etcd
func (k *Runtime) getEtcdCertPath() string {
	return filepath.Join(k.getPKIPath(), "etcd")
}

// /var/lib/sealer/data/my-cluster/rootfs/statics
func (k *Runtime) getStaticFileDir() string {
	return filepath.Join(k.getRootfs(), "statics")
}

func (k *Runtime) getSvcCIDR() string {
	return k.ClusterConfiguration.Networking.ServiceSubnet
}

func (k *Runtime) setCertSANS(certSANS []string) {
	k.ClusterConfiguration.APIServer.CertSANs = strUtils.RemoveDuplicate(append(k.getCertSANS(), certSANS...))
}

func (k *Runtime) getCertSANS() []string {
	return k.ClusterConfiguration.APIServer.CertSANs
}

func (k *Runtime) getDNSDomain() string {
	if k.ClusterConfiguration.Networking.DNSDomain == "" {
		k.ClusterConfiguration.Networking.DNSDomain = "cluster.local"
	}
	return k.ClusterConfiguration.Networking.DNSDomain
}

func (k *Runtime) getAPIServerDomain() string {
	return k.APIServerDomain
}

func (k *Runtime) getKubeVersion() string {
	return k.KubernetesVersion
}

func (k *Runtime) getVIP() net.IP {
	return net.ParseIP(DefaultVIP)
}

func (k *Runtime) getJoinToken() string {
	if k.Discovery.BootstrapToken == nil {
		return ""
	}
	return k.JoinConfiguration.Discovery.BootstrapToken.Token
}

func (k *Runtime) setJoinToken(token string) {
	if k.Discovery.BootstrapToken == nil {
		k.Discovery.BootstrapToken = &v1beta2.BootstrapTokenDiscovery{}
	}
	k.Discovery.BootstrapToken.Token = token
}

func (k *Runtime) getTokenCaCertHash() string {
	if k.Discovery.BootstrapToken == nil || len(k.Discovery.BootstrapToken.CACertHashes) == 0 {
		return ""
	}
	return k.Discovery.BootstrapToken.CACertHashes[0]
}

func (k *Runtime) setTokenCaCertHash(tokenCaCertHash []string) {
	if k.Discovery.BootstrapToken == nil {
		k.Discovery.BootstrapToken = &v1beta2.BootstrapTokenDiscovery{}
	}
	k.Discovery.BootstrapToken.CACertHashes = tokenCaCertHash
}

func (k *Runtime) getCertificateKey() string {
	if k.JoinConfiguration.ControlPlane == nil {
		return ""
	}
	return k.JoinConfiguration.ControlPlane.CertificateKey
}

func (k *Runtime) setInitCertificateKey(certificateKey string) {
	k.CertificateKey = certificateKey
}

func (k *Runtime) setAPIServerEndpoint(endpoint string) {
	k.JoinConfiguration.Discovery.BootstrapToken.APIServerEndpoint = endpoint
}

func (k *Runtime) setInitAdvertiseAddress(advertiseAddress net.IP) {
	k.InitConfiguration.LocalAPIEndpoint.AdvertiseAddress = string(advertiseAddress)
}

func (k *Runtime) setJoinAdvertiseAddress(advertiseAddress net.IP) {
	if k.JoinConfiguration.ControlPlane == nil {
		k.JoinConfiguration.ControlPlane = &v1beta2.JoinControlPlane{}
	}
	k.JoinConfiguration.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = string(advertiseAddress)
}

func (k *Runtime) cleanJoinLocalAPIEndPoint() {
	k.JoinConfiguration.ControlPlane = nil
}

func (k *Runtime) setControlPlaneEndpoint(endpoint string) {
	k.ControlPlaneEndpoint = endpoint
}

func (k *Runtime) setCgroupDriver(cGroup string) {
	k.KubeletConfiguration.CgroupDriver = cGroup
}

func (k *Runtime) setAPIVersion(apiVersion string) {
	k.InitConfiguration.APIVersion = apiVersion
	k.ClusterConfiguration.APIVersion = apiVersion
	k.JoinConfiguration.APIVersion = apiVersion
}

func (k *Runtime) setKubeadmAPIVersion() {
	kv := versionUtils.Version(k.getKubeVersion())
	greatThanKV1150, err := kv.Compare(V1150)
	if err != nil {
		logrus.Errorf("compare kubernetes version failed: %s", err)
	}
	greatThanKV1230, err := kv.Compare(V1230)
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
func (k *Runtime) getCgroupDriverFromShell(node net.IP) (string, error) {
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
		// by default if we get wrong output we set it default systemd
		logrus.Errorf("failed to get nodes [%s] cgroup driver", node)
		driver = DefaultSystemdCgroupDriver
	}
	driver = strings.TrimSpace(driver)
	logrus.Debugf("get nodes [%s] cgroup driver is [%s]", node, driver)
	return driver, nil
}

func (k *Runtime) MergeKubeadmConfig() error {
	if k.getKubeVersion() != "" {
		return nil
	}
	if k.KubeadmConfigFromClusterfile != nil {
		if err := k.LoadFromClusterfile(k.KubeadmConfigFromClusterfile); err != nil {
			return fmt.Errorf("failed to load kubeadm config from clusterfile: %v", err)
		}
	}
	if err := k.Merge(k.getDefaultKubeadmConfig()); err != nil {
		return fmt.Errorf("failed to merge kubeadm config: %v", err)
	}
	k.setKubeadmAPIVersion()
	return nil
}

func (k *Runtime) WaitSSHReady(tryTimes int, hosts ...net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			for i := 0; i < tryTimes; i++ {
				sshClient, err := k.getHostSSHClient(host)
				if err != nil {
					return err
				}
				err = sshClient.Ping(host)
				if err == nil {
					return nil
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			return fmt.Errorf("wait for [%s] ssh ready timeout, ensure that the IP address or password is correct", host)
		})
	}
	return eg.Wait()
}
