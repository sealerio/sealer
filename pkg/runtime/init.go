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
	"path/filepath"
	"strings"
	"sync"

	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

const (
	RemoteCmdCopyStatic            = "mkdir -p %s && cp -f %s %s"
	RemoteApplyYaml                = `echo '%s' | kubectl apply -f -`
	RemoteCmdGetNetworkInterface   = "ls /sys/class/net"
	RemoteCmdExistNetworkInterface = "ip addr show %s | egrep \"%s\" || true"
	WriteKubeadmConfigCmd          = `cd %s && echo '%s' > kubeadm-config.yaml`
	DefaultVIP                     = "10.103.97.2"
	DefaultAPIserverDomain         = "apiserver.cluster.local"
	DefaultRegistryPort            = 5000
	DockerCertDir                  = "/etc/docker/certs.d"
)

func (k *KubeadmRuntime) ConfigKubeadmOnMaster0() error {
	if err := k.LoadFromClusterfile(k.Config.Clusterfile); err != nil {
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
	sshClient, err := k.getHostSSHClient(k.getMaster0IP())
	if err != nil {
		return err
	}
	return sshClient.CmdAsync(k.getMaster0IP(), cmd)
}

func (k *KubeadmRuntime) generateConfigs() ([]byte, error) {
	//getCgroupDriverFromShell need get CRISocket, so after merge
	k.setCgroupDriver(k.getCgroupDriverFromShell(k.getMaster0IP()))
	k.setKubeadmAPIVersion()
	return utils.MarshalConfigsYaml(&k.InitConfiguration,
		&k.ClusterConfiguration,
		&k.KubeletConfiguration,
		&k.KubeProxyConfiguration)
}

func (k *KubeadmRuntime) handleKubeadmConfig() {
	//The configuration set here does not require merge
	k.setInitAdvertiseAddress(k.getMaster0IP())
	k.setControlPlaneEndpoint(fmt.Sprintf("%s:6443", k.getAPIServerDomain()))
	if k.APIServer.ExtraArgs == nil {
		k.APIServer.ExtraArgs = make(map[string]string)
	}
	k.APIServer.ExtraArgs[EtcdServers] = getEtcdEndpointsWithHTTPSPrefix(k.getMasterIPList())
	k.IPVS.ExcludeCIDRs = append(k.KubeProxyConfiguration.IPVS.ExcludeCIDRs, fmt.Sprintf("%s/32", k.getVIP()))
}

//CmdToString is in host exec cmd and replace to spilt str
func (k *KubeadmRuntime) CmdToString(host, cmd, split string) string {
	ssh, err := k.getHostSSHClient(host)
	if err != nil {
		logger.Error("failed to get host ssh client, %s %v", cmd, err)
		return ""
	}
	data, err := ssh.Cmd(host, cmd)
	if err != nil {
		logger.Error("exec remote cmd failed, %s %v", cmd, err)
	}
	if data != nil {
		str := string(data)
		str = strings.ReplaceAll(str, "\r\n", split)
		return str
	}
	return ""
}

func (k *KubeadmRuntime) getRemoteHostName(hostIP string) string {
	hostName := k.CmdToString(hostIP, "hostname", "")
	return strings.ToLower(hostName)
}

func (k *KubeadmRuntime) GenerateCert() error {
	err := cert.GenerateCert(
		k.getPKIPath(),
		k.getEtcdCertPath(),
		k.getCertSANS(),
		k.getMaster0IP(),
		k.getRemoteHostName(k.getMaster0IP()),
		k.getSvcCIDR(),
		k.getDNSDomain(),
	)
	if err != nil {
		return fmt.Errorf("generate certs failed %v", err)
	}
	err = cert.GenerateRegistryCert(k.getCertsDir(), SeaHub)
	if err != nil {
		return err
	}
	err = k.sendNewCertAndKey(k.getMasterIPList()[:1])
	if err != nil {
		return err
	}
	return k.sendRegistryCert(k.getMasterIPList()[:1])
}

func (k *KubeadmRuntime) CreateKubeConfig() error {
	hostname := k.getRemoteHostName(k.getMaster0IP())
	certConfig := cert.Config{
		Path:     k.getPKIPath(),
		BaseName: "ca",
	}

	controlPlaneEndpoint := fmt.Sprintf("https://%s:6443", k.getAPIServerDomain())
	err := cert.CreateJoinControlPlaneKubeConfigFiles(k.getBasePath(),
		certConfig, hostname, controlPlaneEndpoint, "kubernetes")
	if err != nil {
		return fmt.Errorf("generator kubeconfig failed %s", err)
	}
	return nil
}

func (k *KubeadmRuntime) CopyStaticFiles(nodes []string) error {
	errCh := make(chan error, len(nodes))
	defer close(errCh)

	for _, file := range MasterStaticFiles {
		staticFilePath := filepath.Join(k.getStaticFileDir(), file.Name)
		cmdLinkStatic := fmt.Sprintf(RemoteCmdCopyStatic, file.DestinationDir, staticFilePath, filepath.Join(file.DestinationDir, file.Name))
		var wg sync.WaitGroup
		for _, host := range nodes {
			wg.Add(1)
			go func(host string) {
				defer wg.Done()
				ssh, err := k.getHostSSHClient(host)
				if err != nil {
					errCh <- fmt.Errorf("new ssh client failed %v", err)
					return
				}
				err = ssh.CmdAsync(host, cmdLinkStatic)
				if err != nil {
					errCh <- fmt.Errorf("[%s] link static file failed, error:%s", host, err.Error())
				}
			}(host)
		}
		wg.Wait()
	}

	return ReadChanError(errCh)
}

//decode output to join token  hash and key
func (k *KubeadmRuntime) decodeMaster0Output(output []byte) {
	s0 := string(output)
	logger.Debug("decodeOutput: %s", s0)
	slice := strings.Split(s0, "kubeadm join")
	slice1 := strings.Split(slice[1], "Please note")
	logger.Info("join command is: kubeadm join %s", slice1[0])
	k.decodeJoinCmd(slice1[0])
}

//  192.168.0.200:6443 --token 9vr73a.a8uxyaju799qwdjv --discovery-token-ca-cert-hash sha256:7c2e69131a36ae2a042a339b33381c6d0d43887e2de83720eff5359e26aec866 --experimental-control-plane --certificate-key f8902e114ef118304e561c3ecd4d0b543adc226b7a07f675f56564185ffe0c07
func (k *KubeadmRuntime) decodeJoinCmd(cmd string) {
	logger.Debug("[globals]decodeJoinCmd: %s", cmd)
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
	logger.Debug("joinToken: %v\nTokenCaCertHash: %v\nCertificateKey: %v", k.getJoinToken(), k.getTokenCaCertHash(), k.getCertificateKey())
}

//InitMaster0 is
func (k *KubeadmRuntime) InitMaster0() error {
	ssh, err := k.getHostSSHClient(k.getMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get master0 ssh client, %v", err)
	}

	if err := k.SendJoinMasterKubeConfigs([]string{k.getMaster0IP()}, AdminConf, ControllerConf, SchedulerConf, KubeletConf); err != nil {
		return err
	}
	cmdAddEtcHost := fmt.Sprintf(RemoteAddEtcHosts, getAPIServerHost(k.getMaster0IP(), k.getAPIServerDomain()))
	err = ssh.CmdAsync(k.getMaster0IP(), cmdAddEtcHost)
	if err != nil {
		return err
	}

	logger.Info("start to init master0...")
	cmdInit := k.Command(k.getKubeVersion(), InitMaster)

	// TODO skip docker version error check for test
	output, err := ssh.Cmd(k.getMaster0IP(), cmdInit)
	logger.Info("%s", output)
	if err != nil {
		return fmt.Errorf("init master0 failed, error: %s. Please clean and reinstall", err.Error())
	}
	k.decodeMaster0Output(output)
	err = ssh.CmdAsync(k.getMaster0IP(), RemoteCopyKubeConfig)
	if err != nil {
		return err
	}

	return nil
}

func (k *KubeadmRuntime) GetKubectlAndKubeconfig() error {
	if utils.IsFileExist(common.DefaultKubeConfigFile()) {
		return nil
	}
	ssh, err := k.getHostSSHClient(k.getMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get master0 ssh client when get kubbectl and kubeconfig %v", err)
	}

	return GetKubectlAndKubeconfig(ssh, k.getMaster0IP())
}

func (k *KubeadmRuntime) CopyStaticFilesTomasters() error {
	return k.CopyStaticFiles(k.getMasterIPList())
}

func (k *KubeadmRuntime) init(cluster *v2.Cluster) error {
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
			return fmt.Errorf("failed to init master0 %v", err)
		}
	}

	return nil
}
