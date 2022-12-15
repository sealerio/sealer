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

package kubernetes

import (
	"fmt"
	"net"

	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/sealerio/sealer/utils/yaml"
)

func (k *Runtime) joinMasters(newMasters []net.IP, master0 net.IP, kubeadmConfig kubeadm.KubeadmConfig, token v1beta2.BootstrapTokenDiscovery, certKey string) error {
	if len(newMasters) == 0 {
		return nil
	}

	logrus.Infof("%s will be added as master", newMasters)

	if err := k.initKube(newMasters); err != nil {
		return err
	}

	if err := k.copyStaticFiles(newMasters); err != nil {
		return err
	}

	if err := k.sendKubeConfigFilesToMaster(newMasters, kubeadmConfig.KubernetesVersion, AdminConf, ControllerConf, SchedulerConf); err != nil {
		return err
	}

	// TODO only needs send ca?
	if err := k.sendClusterCert(newMasters); err != nil {
		return err
	}

	joinCmd, err := k.Command(kubeadmConfig.KubernetesVersion, master0.String(), JoinMaster, token, certKey)
	if err != nil {
		return fmt.Errorf("failed to get join master command, kubernetes version is %s", kubeadmConfig.KubernetesVersion)
	}
	//set master0 as APIServerEndpoint when join master
	vs := net.JoinHostPort(master0.String(), "6443")
	for _, m := range newMasters {
		logrus.Infof("Start to join %s as master", m)

		hostname, err := k.infra.GetHostName(m)
		if err != nil {
			return err
		}

		kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken = &token
		kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken.APIServerEndpoint = vs
		kubeadmConfig.JoinConfiguration.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = m.String()
		kubeadmConfig.JoinConfiguration.ControlPlane.LocalAPIEndpoint.BindPort = int32(6443)
		kubeadmConfig.JoinConfiguration.ControlPlane.CertificateKey = certKey
		str, err := yaml.MarshalWithDelimiter(kubeadmConfig.JoinConfiguration, kubeadmConfig.KubeletConfiguration)
		if err != nil {
			return err
		}
		cmd := fmt.Sprintf("mkdir -p /etc/kubernetes && echo \"%s\" > %s", str, KubeadmFileYml)
		if err = k.infra.CmdAsync(m, cmd); err != nil {
			return fmt.Errorf("failed to set join kubeadm config on host(%s) with cmd(%s): %v", m, cmd, err)
		}

		if err = k.infra.CmdAsync(m, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), master0.String(), shellcommand.DefaultSealerHostAliasForApiserver)); err != nil {
			return fmt.Errorf("failed to set hosts alias on(%s): %v", m, err)
		}

		certCMD := runtime.RemoteCertCmd(kubeadmConfig.GetCertSANS(), m, hostname, kubeadmConfig.GetSvcCIDR(), "")
		if err = k.infra.CmdAsync(m, certCMD); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", certCMD, m, err)
		}

		if err = k.infra.CmdAsync(m, joinCmd); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", joinCmd, m, err)
		}

		if err = k.infra.CmdAsync(m, "rm -rf .kube/config && mkdir -p /root/.kube && cp /etc/kubernetes/admin.conf /root/.kube/config"); err != nil {
			return err
		}

		logrus.Infof("Succeeded in joining %s as master", m)
	}
	return nil
}
