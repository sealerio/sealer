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
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/alibaba/sealer/utils"

	"github.com/imdario/mergo"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils/ssh"
)

type RuntimeConfig struct {
	Vlog         int
	VIP          string
	RegistryPort string
	// Clusterfile path and name, we needs read kubeadm config from Clusterfile
	Clusterfile     string
	APIServerDomain string

	// TODO move into kubeadm config
	JoinToken       string
	TokenCaCertHash string
}

func newKubeadmRuntime(cluster *v2.Cluster, clusterfile string) (Interface, error) {
	k := &KubeadmRuntime{
		Cluster: cluster,
		RuntimeConfig: &RuntimeConfig{
			Clusterfile:     clusterfile,
			APIServerDomain: DefaultAPIserverDomain,
		},
	}
	// TODO args pre checks
	if err := k.checkList(); err != nil {
		return nil, err
	}

	if logger.IsDebugModel() {
		k.Vlog = 6
	}
	return k, nil
}

// If node has it own ssh config, over write it
func (k *KubeadmRuntime) getSSHClient(hostSSH *v1.SSH) (ssh.Interface, error) {
	if err := mergo.Merge(hostSSH, k.Cluster.Spec.SSH); err != nil {
		return nil, err
	}

	return ssh.NewSSHClient(hostSSH), nil
}

func (k *KubeadmRuntime) checkList() error {
	return k.checkIPList()
}

func (k *KubeadmRuntime) checkIPList() error {
	if len(k.Spec.Hosts) < 1 {
		return fmt.Errorf("master hosts should not < 1, hosts len is %s", len(k.Spec.Hosts))
	}
	if len(k.Spec.Hosts[0].IPS) < 1 {
		return fmt.Errorf("master hosts ip should not < 1, hosts ip len is %s", len(k.Spec.Hosts[0].IPS))
	}
	return nil
}

func (k *KubeadmRuntime) getClusterName() string {
	return k.Cluster.Name
}

func (k *KubeadmRuntime) getHostSSHClient(hostIP string) (ssh.Interface, error) {
	for _, host := range k.Cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			if hostIP == ip {
				return k.getSSHClient(&host.SSH)
			}
		}
	}
	return nil, fmt.Errorf("get host ssh client failed, host ip %s not in hosts ip list", hostIP)
}

func (k *KubeadmRuntime) getRootfs() string {
	return common.DefaultTheClusterRootfsDir(k.Cluster.Name)
}

func (k *KubeadmRuntime) getBasePath() string {
	return path.Join(common.DefaultClusterRootfsDir, k.Cluster.Name)
}

func (k *KubeadmRuntime) getMaster0IP() string {
	// aready check ip list when new the runtime
	return k.Cluster.Spec.Hosts[0].IPS[0]
}

func (k *KubeadmRuntime) getDefaultKubeadmConfig() string {
	return filepath.Join(k.getRootfs(), "etc", "kubeadm.yml")
}

func (k *KubeadmRuntime) getCloudImageDir() string {
	return filepath.Join(common.DefaultMountCloudImageDir(k.Cluster.Name))
}

func (k *KubeadmRuntime) getCertPath() string {
	return path.Join(common.DefaultClusterRootfsDir, k.Cluster.Name, "pki")
}

func (k *KubeadmRuntime) getEtcdCertPath() string {
	return path.Join(common.DefaultClusterRootfsDir, k.Cluster.Name, "pki", "etcd")
}

func (k *KubeadmRuntime) getStaticFileDir() string {
	return path.Join(k.getRootfs(), "statics")
}

func (k *KubeadmRuntime) getSvcCIDR() string {
	return k.ClusterConfiguration.Networking.ServiceSubnet
}

func (k *KubeadmRuntime) getDNSDomain() string {
	return k.ClusterConfiguration.Networking.DNSDomain
}

func (k *KubeadmRuntime) getAPIServerDomain() string {
	return k.RuntimeConfig.APIServerDomain
}

func (k *KubeadmRuntime) getKubeVersion() string {
	return k.KubernetesVersion
}

func (k *KubeadmRuntime) getVIP() string {
	return DefaultVIP
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

func (k *KubeadmRuntime) WaitSSHReady(tryTimes int, hosts ...string) error {
	var err error
	var wg sync.WaitGroup
	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			for i := 0; i < tryTimes; i++ {
				ssh, err := k.getHostSSHClient(host)
				if err != nil {
					return
				}

				err = ssh.Ping(host)
				if err == nil {
					return
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			err = fmt.Errorf("wait for [%s] ssh ready timeout:  %v, ensure that the IP address or password is correct", host, err)
		}(h)
	}
	wg.Wait()
	return err
}
