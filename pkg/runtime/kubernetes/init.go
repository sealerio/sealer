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

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clustercert"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sealerio/sealer/utils/yaml"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	RemoteCmdCopyStatic            = "mkdir -p %s && cp -f %s %s"
	RemoteApplyYaml                = `echo '%s' | kubectl apply -f -`
	RemoteCmdGetNetworkInterface   = "ls /sys/class/net"
	RemoteCmdExistNetworkInterface = "ip addr show %s | egrep \"%s\" || true"
	WriteKubeadmConfigCmd          = `cd %s && echo '%s' > etc/kubeadm.yml`
	DefaultVIP                     = "10.103.97.2"
	DefaultAPIserverDomain         = "apiserver.cluster.local"
	DefaultRegistryPort            = 5000
	DockerCertDir                  = "/etc/docker/certs.d"
)

func (k *Runtime) ConfigKubeadmOnMaster0() error {
	if err := k.LoadFromClusterfile(k.Config.ClusterFileKubeConfig); err != nil {
		return fmt.Errorf("failed to load kubeadm config from clusterfile: %v", err)
	}
	// TODO handle the kubeadm config, like kubeproxy config
	k.handleKubeadmConfig()
	if err := k.KubeadmConfig.Merge(k.getDefaultKubeadmConfig()); err != nil {
		return err
	}
	bs, err := k.generateConfigs()
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf(WriteKubeadmConfigCmd, k.getRootfs(), string(bs))
	sshClient, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return err
	}
	return sshClient.CmdAsync(k.cluster.GetMaster0IP(), cmd)
}

func (k *Runtime) generateConfigs() ([]byte, error) {
	//getCgroupDriverFromShell need get CRISocket, so after merge
	cGroupDriver, err := k.getCgroupDriverFromShell(k.cluster.GetMaster0IP())
	if err != nil {
		return nil, err
	}
	k.setCgroupDriver(cGroupDriver)
	k.setKubeadmAPIVersion()
	return yaml.MarshalWithDelimiter(&k.InitConfiguration,
		&k.ClusterConfiguration,
		&k.KubeletConfiguration,
		&k.KubeProxyConfiguration)
}

func (k *Runtime) handleKubeadmConfig() {
	//The configuration set here does not require merge
	k.setInitAdvertiseAddress(k.cluster.GetMaster0IP())
	k.setControlPlaneEndpoint(fmt.Sprintf("%s:6443", k.getAPIServerDomain()))
	if k.APIServer.ExtraArgs == nil {
		k.APIServer.ExtraArgs = make(map[string]string)
	}
	k.APIServer.ExtraArgs[EtcdServers] = getEtcdEndpointsWithHTTPSPrefix(k.cluster.GetMasterIPList())
	k.IPVS.ExcludeCIDRs = append(k.KubeProxyConfiguration.IPVS.ExcludeCIDRs, fmt.Sprintf("%s/32", k.getVIP()))
}

// CmdToString is in host exec cmd and replace to spilt str
func (k *Runtime) CmdToString(host net.IP, cmd, split string) (string, error) {
	ssh, err := k.getHostSSHClient(host)
	if err != nil {
		return "", fmt.Errorf("failed to get ssh clientof host(%s): %v", host, err)
	}
	return ssh.CmdToString(host, cmd, split)
}

func (k *Runtime) getRemoteHostName(hostIP net.IP) (string, error) {
	hostName, err := k.CmdToString(hostIP, "hostname", "")
	if err != nil {
		return "", err
	}
	if hostName == "" {
		return "", fmt.Errorf("faild to get remote hostname of host(%s)", hostIP)
	}
	return strings.ToLower(hostName), nil
}

func (k *Runtime) GenerateCert() error {
	hostName, err := k.getRemoteHostName(k.cluster.GetMaster0IP())
	if err != nil {
		return err
	}
	err = clustercert.GenerateAllKubernetesCerts(
		k.getPKIPath(),
		k.getEtcdCertPath(),
		hostName,
		k.getSvcCIDR(),
		k.getDNSDomain(),
		k.getCertSANS(),
		k.cluster.GetMaster0IP(),
	)
	if err != nil {
		return fmt.Errorf("failed to generate certs: %v", err)
	}
	err = k.sendNewCertAndKey(k.cluster.GetMasterIPList()[:1])
	if err != nil {
		return err
	}
	err = k.GenerateRegistryCert()
	if err != nil {
		return err
	}
	return k.SendRegistryCert(k.cluster.GetMasterIPList()[:1])
}

func (k *Runtime) GenerateRegistryCert() error {
	return GenerateRegistryCert(k.getCertsDir(), k.RegConfig.Domain)
}

func (k *Runtime) SendRegistryCert(host []net.IP) error {
	err := k.sendRegistryCertAndKey()
	if err != nil {
		return err
	}
	return k.sendRegistryCert(host)
}

func (k *Runtime) CreateKubeConfig() error {
	hostname, err := k.getRemoteHostName(k.cluster.GetMaster0IP())
	if err != nil {
		return err
	}

	controlPlaneEndpoint := fmt.Sprintf("https://%s:6443", k.getAPIServerDomain())
	err = clustercert.CreateJoinControlPlaneKubeConfigFiles(k.getBasePath(), k.getPKIPath(),
		"ca", hostname, controlPlaneEndpoint, "kubernetes")
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig: %s", err)
	}
	return nil
}

func (k *Runtime) CopyStaticFiles(nodes []net.IP) error {
	for _, file := range MasterStaticFiles {
		staticFilePath := filepath.Join(k.getStaticFileDir(), file.Name)
		cmdLinkStatic := fmt.Sprintf(RemoteCmdCopyStatic, file.DestinationDir, staticFilePath, filepath.Join(file.DestinationDir, file.Name))
		eg, _ := errgroup.WithContext(context.Background())
		for _, host := range nodes {
			host := host
			eg.Go(func() error {
				ssh, err := k.getHostSSHClient(host)
				if err != nil {
					return fmt.Errorf("failed to get ssh client of host(%s): %v", host, err)
				}
				err = ssh.CmdAsync(host, cmdLinkStatic)
				if err != nil {
					return fmt.Errorf("[%s] failed to link static file: %s", host, err.Error())
				}
				return err
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}
	return nil
}

// decode output to join token hash and key
func (k *Runtime) decodeMaster0Output(output []byte) {
	s0 := string(output)
	logrus.Debugf("decodeOutput: %s", s0)
	slice := strings.Split(s0, "kubeadm join")
	slice1 := strings.Split(slice[1], "Please note")
	logrus.Infof("join command is: kubeadm join %s", slice1[0])
	k.decodeJoinCmd(slice1[0])
}

// 192.168.0.200:6443 --token 9vr73a.a8uxyaju799qwdjv --discovery-token-ca-cert-hash sha256:7c2e69131a36ae2a042a339b33381c6d0d43887e2de83720eff5359e26aec866 --experimental-control-plane --certificate-key f8902e114ef118304e561c3ecd4d0b543adc226b7a07f675f56564185ffe0c07
func (k *Runtime) decodeJoinCmd(cmd string) {
	logrus.Debugf("[globals]decodeJoinCmd: %s", cmd)
	stringSlice := strings.Split(cmd, " ")

	for i, r := range stringSlice {
		// upstream error, delete \t, \\, \n, space.
		r = strings.ReplaceAll(r, "\t", "")
		r = strings.ReplaceAll(r, "\n", "")
		r = strings.ReplaceAll(r, "\\", "")
		r = strings.TrimSpace(r)
		if strings.Contains(r, "--token") {
			k.setJoinToken(stringSlice[i+1])
		}
		if strings.Contains(r, "--discovery-token-ca-cert-hash") {
			k.setTokenCaCertHash([]string{stringSlice[i+1]})
		}
		if strings.Contains(r, "--certificate-key") {
			k.setInitCertificateKey(stringSlice[i+1][:64])
		}
	}
	logrus.Debugf("joinToken: %v\nTokenCaCertHash: %v\nCertificateKey: %v", k.getJoinToken(), k.getTokenCaCertHash(), k.getCertificateKey())
}

// InitMaster0 is using kubeadm init to start up the cluster master0.
func (k *Runtime) InitMaster0() error {
	client, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get ssh client of master0(%s): %v", k.cluster.GetMaster0IP(), err)
	}

	if err := k.SendJoinMasterKubeConfigs([]net.IP{k.cluster.GetMaster0IP()}, AdminConf, ControllerConf, SchedulerConf, KubeletConf); err != nil {
		return err
	}
	apiServerHost := getAPIServerHost(k.cluster.GetMaster0IP(), k.getAPIServerDomain())
	cmdAddEtcHost := fmt.Sprintf(RemoteAddEtcHosts, apiServerHost, apiServerHost)
	err = client.CmdAsync(k.cluster.GetMaster0IP(), cmdAddEtcHost)
	if err != nil {
		return err
	}

	logrus.Info("start to init master0...")
	cmdInit := k.Command(k.getKubeVersion(), InitMaster)

	// TODO skip docker version error check for test
	output, err := client.Cmd(k.cluster.GetMaster0IP(), cmdInit)
	if err != nil {
		return fmt.Errorf("failed to init master0: %s. Please clean and reinstall", err)
	}
	k.decodeMaster0Output(output)
	err = client.CmdAsync(k.cluster.GetMaster0IP(), RemoteCopyKubeConfig)
	if err != nil {
		return err
	}

	if client.(*ssh.SSH).User != common.ROOT {
		err = client.CmdAsync(k.cluster.GetMaster0IP(), RemoteNonRootCopyKubeConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Runtime) GetKubectlAndKubeconfig() error {
	if osi.IsFileExist(common.DefaultKubeConfigFile()) {
		return nil
	}
	client, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get ssh client of master0(%s) when get kubbectl and kubeconfig: %v", k.cluster.GetMaster0IP(), err)
	}

	return GetKubectlAndKubeconfig(client, k.cluster.GetMaster0IP(), k.getImageMountDir())
}

func (k *Runtime) CopyStaticFilesTomasters() error {
	return k.CopyStaticFiles(k.cluster.GetMasterIPList())
}

func (k *Runtime) init() error {
	pipeline := []func() error{
		k.ConfigKubeadmOnMaster0,
		k.GenerateCert,
		k.CreateKubeConfig,
		k.CopyStaticFilesTomasters,
		k.ApplyRegistry,
		k.InitMaster0,
		k.GetKubectlAndKubeconfig,
	}

	for _, f := range pipeline {
		if err := f(); err != nil {
			return fmt.Errorf("failed to init cluster: %v", err)
		}
	}

	return nil
}
