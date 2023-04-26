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

	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/sealerio/sealer/utils/yaml"
)

func (k *Runtime) joinMasters(newMasters []net.IP, master0 net.IP, kubeadmConfig kubeadm.KubeadmConfig, token v1beta3.BootstrapTokenDiscovery, certKey string) error {
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

	if err := k.sendKubeConfigFilesToMaster(newMasters, AdminConf, ControllerConf, SchedulerConf); err != nil {
		return err
	}

	if err := k.sendKubeadmFile(newMasters); err != nil {
		return err
	}

	// TODO only needs send ca?
	if err := k.sendClusterCert(newMasters); err != nil {
		return err
	}

	//set master0 as APIServerEndpoint when join master
	vs := net.JoinHostPort(master0.String(), "6443")
	for _, m := range newMasters {
		logrus.Infof("start to join %s as master", m)

		joinCmd, err := k.Command(JoinMaster, k.getNodeNameOverride(m))
		if err != nil {
			return fmt.Errorf("failed to get join master command: %v", err)
		}

		hostname, err := k.infra.GetHostName(m)
		if err != nil {
			return err
		}

		if output, err := k.infra.CmdToString(m, nil, GetCustomizeCRISocket, ""); err == nil && output != "" {
			kubeadmConfig.JoinConfiguration.NodeRegistration.CRISocket = output
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
		if err = k.infra.CmdAsync(m, nil, cmd); err != nil {
			return fmt.Errorf("failed to set join kubeadm config on host(%s) with cmd(%s): %v", m, cmd, err)
		}

		if err = k.infra.CmdAsync(m, nil, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), master0.String())); err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}

		certCMD := runtime.RemoteCertCmd(kubeadmConfig.GetCertSANS(), m, hostname, kubeadmConfig.GetSvcCIDR(), "")
		if err = k.infra.CmdAsync(m, nil, certCMD); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", certCMD, m, err)
		}

		if err = k.infra.CmdAsync(m, nil, joinCmd); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", joinCmd, m, err)
		}

		if err = k.infra.CmdAsync(m, nil, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), m.String())); err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}

		if err = k.infra.CmdAsync(m, nil, "rm -rf .kube/config && mkdir -p /root/.kube && cp /etc/kubernetes/admin.conf /root/.kube/config"); err != nil {
			return err
		}

		// At beginning, we set APIServerDomain direct to master0 and then kubeadm start scheduler and kcm, then we reset
		// the APIServerDomain to the master itself, but scheduler and kcm already load the domain info and will not reload.
		// So, we need restart them after reset the APIServerDomain.
		if err = k.infra.CmdAsync(m, nil, "mv /etc/kubernetes/manifests/kube-scheduler.yaml /tmp/ && mv /tmp/kube-scheduler.yaml /etc/kubernetes/manifests/"); err != nil {
			return err
		}
		if err = k.infra.CmdAsync(m, nil, "mv /etc/kubernetes/manifests/kube-controller-manager.yaml /tmp/ && mv /tmp/kube-controller-manager.yaml /etc/kubernetes/manifests/"); err != nil {
			return err
		}

		logrus.Infof("succeeded in joining %s as master", m)
	}
	return nil
}
