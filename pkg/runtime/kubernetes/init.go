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
	"os"
	"path"
	"path/filepath"
	"strings"

	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clustercert"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/sealerio/sealer/utils/yaml"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) initKubeadmConfig(masters []net.IP) (kubeadm.KubeadmConfig, error) {
	extraSANsStr := k.infra.GetClusterEnv()[common.EnvCertSANs]
	var extraSANs []string
	if extraSANsStr != "" {
		extraSANs = strings.Split(extraSANsStr, ",")
	}
	conf, err := kubeadm.NewKubeadmConfig(
		k.Config.KubeadmConfigFromClusterFile,
		k.getDefaultKubeadmConfig(),
		masters,
		k.getAPIServerDomain(),
		k.Config.containerRuntimeInfo.CgroupDriver,
		k.Config.RegistryInfo.URL,
		k.getAPIServerVIP(), extraSANs)
	if err != nil {
		return kubeadm.KubeadmConfig{}, err
	}

	if output, err := k.infra.CmdToString(masters[0], nil, GetCustomizeCRISocket, ""); err == nil && output != "" {
		conf.InitConfiguration.NodeRegistration.CRISocket = output
	}

	bs, err := yaml.MarshalWithDelimiter(&conf.InitConfiguration,
		&conf.ClusterConfiguration,
		&conf.KubeletConfiguration,
		&conf.KubeProxyConfiguration,
		&conf.JoinConfiguration)
	if err != nil {
		return kubeadm.KubeadmConfig{}, err
	}

	localTmpFile := "/tmp/kubeadm.yaml"
	if err = os.WriteFile(localTmpFile, bs, 0600); err != nil {
		return kubeadm.KubeadmConfig{}, err
	}

	if err = k.infra.Copy(masters[0], localTmpFile, KubeadmFileYml); err != nil {
		return kubeadm.KubeadmConfig{}, err
	}
	if err = k.infra.Copy(masters[0], localTmpFile, path.Join(k.infra.GetClusterRootfsPath(), "kubeadm.yaml")); err != nil {
		return kubeadm.KubeadmConfig{}, err
	}

	if err = os.Remove(localTmpFile); err != nil {
		return kubeadm.KubeadmConfig{}, err
	}

	return conf, nil
}

func (k *Runtime) generateCert(kubeadmConf kubeadm.KubeadmConfig, master0 net.IP) error {
	hostName, err := k.infra.GetHostName(master0)
	if err != nil {
		return err
	}

	return clustercert.GenerateAllKubernetesCerts(
		k.getPKIPath(),
		k.getEtcdCertPath(),
		hostName,
		kubeadmConf.GetSvcCIDR(),
		kubeadmConf.GetDNSDomain(),
		kubeadmConf.GetCertSANS(),
		master0,
	)
}

func (k *Runtime) createKubeConfig(master0 net.IP) error {
	hostName, err := k.infra.GetHostName(master0)
	if err != nil {
		return err
	}
	if nno := k.getNodeNameOverride(master0); nno != "" {
		hostName = nno
	}

	controlPlaneEndpoint := fmt.Sprintf("https://%s", net.JoinHostPort(k.getAPIServerDomain(), "6443"))

	return clustercert.CreateJoinControlPlaneKubeConfigFiles(k.infra.GetClusterRootfsPath(), k.getPKIPath(),
		"ca", hostName, controlPlaneEndpoint, "kubernetes")
}

func (k *Runtime) copyStaticFiles(nodes []net.IP) error {
	for _, file := range MasterStaticFiles {
		staticFilePath := filepath.Join(k.getStaticFileDir(), file.Name)
		cmdLinkStatic := fmt.Sprintf("mkdir -p %s && cp -f %s %s", file.DestinationDir, staticFilePath, filepath.Join(file.DestinationDir, file.Name))
		eg, _ := errgroup.WithContext(context.Background())
		for _, host := range nodes {
			h := host
			eg.Go(func() error {
				if err := k.infra.CmdAsync(h, nil, cmdLinkStatic); err != nil {
					return fmt.Errorf("[%s] failed to link static file: %s", h, err.Error())
				}

				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}
	return nil
}

// initMaster0 is using kubeadm init to start up the cluster master0.
func (k *Runtime) initMaster0(master0 net.IP) (v1beta3.BootstrapTokenDiscovery, string, error) {
	if err := k.initKube([]net.IP{master0}); err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}

	if err := k.sendClusterCert([]net.IP{master0}); err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}

	if err := k.sendKubeConfigFilesToMaster([]net.IP{master0}, AdminConf, ControllerConf, SchedulerConf, KubeletConf); err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}

	if err := k.infra.CmdAsync(master0, nil, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), master0.String())); err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
	}

	cmdInit, err := k.Command(InitMaster, k.getNodeNameOverride(master0))
	if err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}
	logrus.Info("start to init master0...")

	// TODO skip docker version error check for test
	output, err := k.infra.Cmd(master0, nil, cmdInit)
	if err != nil {
		_, wErr := common.StdOut.WriteString(string(output))
		if wErr != nil {
			return v1beta3.BootstrapTokenDiscovery{}, "", err
		}
		return v1beta3.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to init master0: %s. Please clean and reinstall", err)
	}

	if err = k.infra.CmdAsync(master0, nil, "rm -rf .kube/config && mkdir -p /root/.kube && cp /etc/kubernetes/admin.conf /root/.kube/config"); err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}

	token, certKey := k.decodeMaster0Output(output)

	return token, certKey, nil
}

// decode output to join token hash and key
func (k *Runtime) decodeMaster0Output(output []byte) (v1beta3.BootstrapTokenDiscovery, string) {
	s0 := string(output)
	logrus.Debugf("decodeOutput: %s", s0)
	slice := strings.Split(s0, "kubeadm join")
	slice1 := strings.Split(slice[1], "Please note")
	logrus.Infof("join command is: kubeadm join %s", slice1[0])

	return k.decodeJoinCmd(slice1[0])
}

// 192.168.0.200:6443 --token 9vr73a.a8uxyaju799qwdjv --discovery-token-ca-cert-hash sha256:7c2e69131a36ae2a042a339b33381c6d0d43887e2de83720eff5359e26aec866 --experimental-control-plane --certificate-key f8902e114ef118304e561c3ecd4d0b543adc226b7a07f675f56564185ffe0c07
func (k *Runtime) decodeJoinCmd(cmd string) (v1beta3.BootstrapTokenDiscovery, string) {
	logrus.Debugf("[globals]decodeJoinCmd: %s", cmd)
	stringSlice := strings.Split(cmd, " ")

	token := v1beta3.BootstrapTokenDiscovery{}
	var certKey string

	for i, r := range stringSlice {
		// upstream error, delete \t, \\, \n, space.
		r = strings.ReplaceAll(r, "\t", "")
		r = strings.ReplaceAll(r, "\n", "")
		r = strings.ReplaceAll(r, "\\", "")
		r = strings.TrimSpace(r)
		if strings.Contains(r, "--token") {
			token.Token = stringSlice[i+1]
		}
		if strings.Contains(r, "--discovery-token-ca-cert-hash") {
			token.CACertHashes = []string{stringSlice[i+1]}
		}
		if strings.Contains(r, "--certificate-key") {
			certKey = stringSlice[i+1][:64]
		}
	}

	return token, certKey
}

// initKube do some initialize kubelet works, such as configuring the host environment, initializing the kubelet service, and so on.
func (k *Runtime) initKube(hosts []net.IP) error {
	initKubeletCmd := fmt.Sprintf("cd %s && export RegistryURL=%s && bash %s", filepath.Join(k.infra.GetClusterRootfsPath(), "scripts"), k.Config.RegistryInfo.URL, "init-kube.sh")
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			if err := k.infra.CmdAsync(host, nil, initKubeletCmd); err != nil {
				return fmt.Errorf("failed to init Kubelet Service on (%s): %s", host, err.Error())
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) sendClusterCert(hosts []net.IP) error {
	f := func(host net.IP) error {
		if err := k.infra.Copy(host, k.getPKIPath(), clustercert.KubeDefaultCertPath); err != nil {
			return fmt.Errorf("failed to copy cluster cert: %v", err)
		}
		if err := k.infra.Copy(host, k.getPKIPath(), k.getPKIPath()); err != nil {
			return fmt.Errorf("failed to copy cluster cert: %v", err)
		}
		return nil
	}

	return k.infra.Execute(hosts, f)
}

func (k *Runtime) sendKubeadmFile(hosts []net.IP) error {
	f := func(host net.IP) error {
		if err := k.infra.Copy(host, path.Join(k.infra.GetClusterRootfsPath(), "kubeadm.yaml"), path.Join(k.infra.GetClusterRootfsPath(), "kubeadm.yaml")); err != nil {
			return fmt.Errorf("failed to copy kubeadm file: %v", err)
		}
		return nil
	}

	return k.infra.Execute(hosts, f)
}

func (k *Runtime) sendKubeConfigFilesToMaster(masters []net.IP, files ...string) error {
	for _, kubeFile := range files {
		src := filepath.Join(k.infra.GetClusterRootfsPath(), kubeFile)
		dest := filepath.Join(clustercert.KubernetesConfigDir, kubeFile)

		f := func(host net.IP) error {
			if err := k.infra.Copy(host, src, dest); err != nil {
				return fmt.Errorf("failed to copy cluster kubeconfig file : %v", err)
			}
			if err := k.infra.Copy(host, src, src); err != nil {
				return fmt.Errorf("failed to copy cluster kubeconfig file : %v", err)
			}
			return nil
		}
		if err := k.infra.Execute(masters, f); err != nil {
			return err
		}
	}

	return nil
}

func (k *Runtime) getJoinTokenHashAndKey(master0 net.IP) (v1beta3.BootstrapTokenDiscovery, string, error) {
	cmd := fmt.Sprintf(`kubeadm init phase upload-certs --upload-certs -v %d`, k.Config.Vlog)

	output, err := k.infra.CmdToString(master0, nil, cmd, "\r\n")
	if err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", err
	}
	logrus.Debugf("[globals]decodeCertCmd: %s", output)
	slice := strings.Split(output, "Using certificate key:")
	if len(slice) != 2 {
		return v1beta3.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to get certifacate key: %s", slice)
	}
	key := strings.Replace(slice[1], "\r\n", "", -1)
	certKey := strings.Replace(key, "\n", "", -1)

	cmd = fmt.Sprintf("kubeadm token create --print-join-command -v %d", k.Config.Vlog)

	out, err := k.infra.Cmd(master0, nil, cmd)
	if err != nil {
		return v1beta3.BootstrapTokenDiscovery{}, "", fmt.Errorf("failed to create kubeadm join token: %v", err)
	}

	token, certKey2 := k.decodeMaster0Output(out)

	if certKey == "" {
		certKey = certKey2
	}

	return token, certKey, nil
}
