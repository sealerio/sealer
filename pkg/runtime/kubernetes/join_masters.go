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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config/v1beta2"
	"github.com/sealerio/sealer/utils/shellcommand"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sealerio/sealer/pkg/clustercert"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/utils/yaml"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) sendKubeConfigFile(hosts []net.IP, kubeFile string) error {
	absKubeFile := fmt.Sprintf("%s/%s", clustercert.KubernetesConfigDir, kubeFile)
	sealerKubeFile := fmt.Sprintf("%s/%s", k.infra.GetClusterRootfs(), kubeFile)

	return k.sendFileToHosts(hosts, sealerKubeFile, absKubeFile)
}

func (k *Runtime) sendNewCertAndKey(hosts []net.IP) error {
	return k.sendFileToHosts(hosts, k.getPKIPath(), clustercert.KubeDefaultCertPath)
}

func (k *Runtime) sendFileToHosts(Hosts []net.IP, src, dst string) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, n := range Hosts {
		node := n
		eg.Go(func() error {
			return k.infra.Copy(node, src, dst)
		})
	}
	return eg.Wait()
}

func (k *Runtime) ReplaceKubeConfigV1991V1992(kubeVersion string, masters []net.IP) error {
	// fix > 1.19.1 kube-controller-manager and kube-scheduler use the LocalAPIEndpoint instead of the ControlPlaneEndpoint.
	if kubeVersion == kubeadm_config.V1991 || kubeVersion == kubeadm_config.V1992 {
		for _, v := range masters {
			cmd := fmt.Sprintf(RemoteReplaceKubeConfig, KUBESCHEDULERCONFIGFILE, v, KUBECONTROLLERCONFIGFILE, v, KUBESCHEDULERCONFIGFILE)

			if err := k.infra.CmdAsync(v, cmd); err != nil {
				return fmt.Errorf("failed to replace kube config on %s: %v ", v, err)
			}
		}
	}

	return nil
}

func (k *Runtime) SendJoinMasterKubeConfigs(masters []net.IP, kubeVersion string, files ...string) error {
	for _, f := range files {
		if err := k.sendKubeConfigFile(masters, f); err != nil {
			return err
		}
	}

	if err := k.ReplaceKubeConfigV1991V1992(kubeVersion, masters); err != nil {
		logrus.Warningf("failed to set kubernetes v1.19.1 v1.19.2 kube config: %v", err)
	}

	return nil
}

func (k *Runtime) joinMasters(newMasters []net.IP, master0 net.IP, kubeadmConfig kubeadm_config.KubeadmConfig, token v1beta2.BootstrapTokenDiscovery, certKey string) error {
	if len(newMasters) == 0 {
		return nil
	}

	logrus.Infof("%s will be added as master", newMasters)

	if err := k.initKube(newMasters); err != nil {
		return err
	}

	if err := k.CopyStaticFiles(newMasters); err != nil {
		return err
	}

	if err := k.SendJoinMasterKubeConfigs(newMasters, kubeadmConfig.KubernetesVersion, AdminConf, ControllerConf, SchedulerConf); err != nil {
		return err
	}

	// TODO only needs send ca?
	if err := k.sendNewCertAndKey(newMasters); err != nil {
		return err
	}

	joinCmd, err := k.Command(kubeadmConfig.KubernetesVersion, master0.String(), JoinMaster, token, certKey)
	if err != nil {
		return fmt.Errorf("failed to get join master command, kubernetes version is %s", kubeadmConfig.KubernetesVersion)
	}

	for _, m := range newMasters {
		logrus.Infof("Start to join %s as master", m)

		hostname, err := k.infra.GetHostName(m)
		if err != nil {
			return err
		}

		if err := kubeadmConfig.JoinConfiguration.SetJoinAdvertiseAddress(m); err != nil {
			return err
		}
		kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken = &token
		kubeadmConfig.JoinConfiguration.ControlPlane.CertificateKey = certKey
		str, err := yaml.MarshalWithDelimiter(kubeadmConfig.JoinConfiguration, kubeadmConfig.KubeletConfiguration)
		if err != nil {
			return err
		}
		cmd := fmt.Sprintf("echo \"%s\" > %s", str, KubeadmFileYml)
		if err := k.infra.CmdAsync(m, cmd); err != nil {
			return fmt.Errorf("failed to set join kubeadm config on host(%s) with cmd(%s): %v", m, cmd, err)
		}

		if err := k.infra.CmdAsync(m, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), m.String())); err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}

		certCMD := runtime.RemoteCertCmd(kubeadmConfig.GetCertSANS(), m, hostname, kubeadmConfig.GetSvcCIDR(), "")
		if err := k.infra.CmdAsync(m, certCMD); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", certCMD, m, err)
		}

		if err := k.infra.CmdAsync(m, joinCmd); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", joinCmd, m, err)
		}

		logrus.Infof("Succeeded in joining %s as master", m)
	}

	return nil
}

func (k *Runtime) getJoinTokenHashAndKey(master0 net.IP) (v1beta2.BootstrapTokenDiscovery, string, error) {
	cmd := fmt.Sprintf(`kubeadm init phase upload-certs --upload-certs -v %d`, k.Config.Vlog)

	output, err := k.infra.CmdToString(master0, cmd, "\r\n")
	if err != nil {
		return v1beta2.BootstrapTokenDiscovery{}, "", err
	}
	logrus.Debugf("[globals]decodeCertCmd: %s", output)
	slice := strings.Split(output, "Using certificate key:")
	if len(slice) != 2 {
		return v1beta2.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to get certifacate key: %s", slice)
	}
	key := strings.Replace(slice[1], "\r\n", "", -1)
	certKey := strings.Replace(key, "\n", "", -1)

	cmd = fmt.Sprintf("kubeadm token create --print-join-command -v %d", k.Config.Vlog)

	out, err := k.infra.Cmd(master0, cmd)
	if err != nil {
		return v1beta2.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to create kubeadm join token: %v", err)
	}

	token, certKey2 := k.decodeMaster0Output(out)

	if certKey == "" {
		certKey = certKey2
	}

	return token, certKey, nil
}

// dumpKubeConfigIntoCluster save AdminKubeConf to cluster as secret resource.
func (k *Runtime) dumpKubeConfigIntoCluster(master0 net.IP) error {
	driver, err := k.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}

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
