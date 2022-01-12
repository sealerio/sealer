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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sealer/pkg/runtime/kubeadm_types/v1beta2"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils/ssh"
)

type Config struct {
	Vlog         int
	VIP          string
	RegistryPort string
	// Clusterfile: the absolute path, we need to read kubeadm config from Clusterfile
	Clusterfile     string
	APIServerDomain string
}

func newKubeadmRuntime(cluster *v2.Cluster, clusterfile string) (Interface, error) {
	k := &KubeadmRuntime{
		Cluster: cluster,
		Config: &Config{
			Clusterfile:     clusterfile,
			APIServerDomain: DefaultAPIserverDomain,
		},
		KubeadmConfig: &KubeadmConfig{},
	}
	k.setCertSANS(append([]string{"127.0.0.1", k.getAPIServerDomain(), k.getVIP()}, k.getMasterIPList()...))
	// TODO args pre checks
	if err := k.checkList(); err != nil {
		return nil, err
	}

	if logger.IsDebugModel() {
		k.Vlog = 6
	}
	return k, nil
}

func (k *KubeadmRuntime) checkList() error {
	if len(k.Spec.Hosts) == 0 {
		return fmt.Errorf("master hosts cannot be empty")
	}
	if len(k.Spec.Hosts[0].IPS) == 0 {
		return fmt.Errorf("master hosts ip cannot be empty")
	}
	return nil
}

func (k *KubeadmRuntime) getClusterName() string {
	return k.Cluster.Name
}

func (k *KubeadmRuntime) getMaster0IP() string {
	// already check ip list when new the runtime
	return k.Cluster.Spec.Hosts[0].IPS[0]
}
func (k *KubeadmRuntime) getClusterMetadata() (*Metadata, error) {
	metadata := &Metadata{}
	if k.getKubeVersion() == "" {
		if err := k.MergeKubeadmConfig(); err != nil {
			return nil, err
		}
	}
	metadata.Version = k.getKubeVersion()
	return metadata, nil
}

func (k *KubeadmRuntime) getDefaultRegistryPort() int {
	return DefaultRegistryPort
}

func (k *KubeadmRuntime) getHostSSHClient(hostIP string) (ssh.Interface, error) {
	return ssh.GetHostSSHClient(hostIP, k.Cluster)
}

// /var/lib/sealer/data/my-cluster
func (k *KubeadmRuntime) getBasePath() string {
	return common.DefaultClusterBaseDir(k.getClusterName())
}

// /var/lib/sealer/data/my-cluster/rootfs
func (k *KubeadmRuntime) getRootfs() string {
	return common.DefaultTheClusterRootfsDir(k.getClusterName())
}

// /var/lib/sealer/data/my-cluster/mount
func (k *KubeadmRuntime) getImageMountDir() string {
	return common.DefaultMountCloudImageDir(k.getClusterName())
}

// /var/lib/sealer/data/my-cluster/certs
func (k *KubeadmRuntime) getCertsDir() string {
	return common.TheDefaultClusterCertDir(k.getClusterName())
}

// /var/lib/sealer/data/my-cluster/pki
func (k *KubeadmRuntime) getPKIPath() string {
	return common.TheDefaultClusterPKIDir(k.getClusterName())
}

// /var/lib/sealer/data/my-cluster/mount/etc/kubeadm.yml
func (k *KubeadmRuntime) getDefaultKubeadmConfig() string {
	return filepath.Join(k.getImageMountDir(), "etc", "kubeadm.yml")
}

// /var/lib/sealer/data/my-cluster/pki/etcd
func (k *KubeadmRuntime) getEtcdCertPath() string {
	return filepath.Join(k.getPKIPath(), "etcd")
}

// /var/lib/sealer/data/my-cluster/rootfs/statics
func (k *KubeadmRuntime) getStaticFileDir() string {
	return filepath.Join(k.getRootfs(), "statics")
}

func (k *KubeadmRuntime) getSvcCIDR() string {
	return k.ClusterConfiguration.Networking.ServiceSubnet
}

func (k *KubeadmRuntime) setCertSANS(certSANS []string) {
	k.ClusterConfiguration.APIServer.CertSANs = utils.RemoveDuplicate(append(k.getCertSANS(), certSANS...))
}

func (k *KubeadmRuntime) getCertSANS() []string {
	return k.ClusterConfiguration.APIServer.CertSANs
}

func (k *KubeadmRuntime) getDNSDomain() string {
	if k.ClusterConfiguration.Networking.DNSDomain == "" {
		k.ClusterConfiguration.Networking.DNSDomain = "cluster.local"
	}
	return k.ClusterConfiguration.Networking.DNSDomain
}

func (k *KubeadmRuntime) getAPIServerDomain() string {
	return k.Config.APIServerDomain
}

func (k *KubeadmRuntime) getKubeVersion() string {
	return k.KubernetesVersion
}

func (k *KubeadmRuntime) getVIP() string {
	return DefaultVIP
}

func (k *KubeadmRuntime) getJoinToken() string {
	if k.Discovery.BootstrapToken == nil {
		return ""
	}
	return k.JoinConfiguration.Discovery.BootstrapToken.Token
}

func (k *KubeadmRuntime) setJoinToken(token string) {
	if k.Discovery.BootstrapToken == nil {
		k.Discovery.BootstrapToken = &v1beta2.BootstrapTokenDiscovery{}
	}
	k.Discovery.BootstrapToken.Token = token
}

func (k *KubeadmRuntime) getTokenCaCertHash() string {
	if k.Discovery.BootstrapToken == nil || len(k.Discovery.BootstrapToken.CACertHashes) == 0 {
		return ""
	}
	return k.Discovery.BootstrapToken.CACertHashes[0]
}

func (k *KubeadmRuntime) setTokenCaCertHash(tokenCaCertHash []string) {
	if k.Discovery.BootstrapToken == nil {
		k.Discovery.BootstrapToken = &v1beta2.BootstrapTokenDiscovery{}
	}
	k.Discovery.BootstrapToken.CACertHashes = tokenCaCertHash
}

func (k *KubeadmRuntime) getCertificateKey() string {
	if k.JoinConfiguration.ControlPlane == nil {
		return ""
	}
	return k.JoinConfiguration.ControlPlane.CertificateKey
}

func (k *KubeadmRuntime) setInitCertificateKey(certificateKey string) {
	k.CertificateKey = certificateKey
}

func (k *KubeadmRuntime) setAPIServerEndpoint(endpoint string) {
	k.JoinConfiguration.Discovery.BootstrapToken.APIServerEndpoint = endpoint
}

func (k *KubeadmRuntime) setInitAdvertiseAddress(advertiseAddress string) {
	k.InitConfiguration.LocalAPIEndpoint.AdvertiseAddress = advertiseAddress
}

func (k *KubeadmRuntime) setJoinAdvertiseAddress(advertiseAddress string) {
	if k.JoinConfiguration.ControlPlane == nil {
		k.JoinConfiguration.ControlPlane = &v1beta2.JoinControlPlane{}
	}
	k.JoinConfiguration.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = advertiseAddress
}

func (k *KubeadmRuntime) cleanJoinLocalAPIEndPoint() {
	k.JoinConfiguration.ControlPlane = nil
}

func (k *KubeadmRuntime) setControlPlaneEndpoint(endpoint string) {
	k.ControlPlaneEndpoint = endpoint
}

func (k *KubeadmRuntime) setCgroupDriver(cGroup string) {
	k.KubeletConfiguration.CgroupDriver = cGroup
}

func (k *KubeadmRuntime) getMasterIPList() (masters []string) {
	return k.getHostsIPByRole(common.MASTER)
}

func (k *KubeadmRuntime) getNodesIPList() (nodes []string) {
	return k.getHostsIPByRole(common.NODE)
}

func (k *KubeadmRuntime) getHostsIPByRole(role string) (nodes []string) {
	for _, host := range k.Spec.Hosts {
		if utils.InList(role, host.Roles) {
			nodes = append(nodes, host.IPS...)
		}
	}

	return
}

func getEtcdEndpointsWithHTTPSPrefix(masters []string) string {
	var tmpSlice []string
	for _, ip := range masters {
		tmpSlice = append(tmpSlice, fmt.Sprintf("https://%s:2379", utils.GetHostIP(ip)))
	}
	return strings.Join(tmpSlice, ",")
}

func (k *KubeadmRuntime) WaitSSHReady(tryTimes int, hosts ...string) error {
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
