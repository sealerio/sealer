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
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/cert"
	"github.com/alibaba/sealer/pkg/command"
	"github.com/alibaba/sealer/pkg/ipvs"
	"github.com/alibaba/sealer/utils"
)

const (
	V1991 = "v1.19.1"
	V1992 = "v1.19.2"
	V1150 = "v1.15.0"
	V1200 = "v1.20.0"
	V1230 = "v1.23.0"
)

const (
	RemoteAddEtcHosts       = "cat /etc/hosts |grep '%s' || echo '%s' >> /etc/hosts"
	RemoteUpdateEtcHosts    = `sed "s/%s/%s/g" < /etc/hosts > hosts && cp -f hosts /etc/hosts`
	RemoteCopyKubeConfig    = `rm -rf .kube/config && mkdir -p /root/.kube && cp /etc/kubernetes/admin.conf /root/.kube/config`
	RemoteReplaceKubeConfig = `grep -qF "apiserver.cluster.local" %s  && sed -i 's/apiserver.cluster.local/%s/' %s && sed -i 's/apiserver.cluster.local/%s/' %s`
	RemoteJoinMasterConfig  = `echo "%s" > %s/etc/kubeadm.yml`
	InitMaster115Lower      = `kubeadm init --config=%s/etc/kubeadm.yml --experimental-upload-certs`
	JoinMaster115Lower      = "kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s --experimental-control-plane --certificate-key %s"
	JoinNode115Lower        = "kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s"
	InitMaser115Upper       = `kubeadm init --config=%s/etc/kubeadm.yml --upload-certs`
	JoinMaster115Upper      = "kubeadm join --config=%s/etc/kubeadm.yml"
	JoinNode115Upper        = "kubeadm join --config=%s/etc/kubeadm.yml"
	RemoveKubeConfig        = "rm -rf /usr/bin/kube* && rm -rf ~/.kube/"
	RemoteCleanMasterOrNode = `if which kubeadm;then kubeadm reset -f %s;fi && \
modprobe -r ipip  && lsmod && \
rm -rf /etc/kubernetes/ && \
rm -rf /etc/systemd/system/kubelet.service.d && rm -rf /etc/systemd/system/kubelet.service && \
rm -rf /usr/bin/kubeadm && rm -rf /usr/bin/kubelet-pre-start.sh && \
rm -rf /usr/bin/kubelet && rm -rf /usr/bin/crictl && \
rm -rf /etc/cni && rm -rf /opt/cni && \
rm -rf /var/lib/etcd && rm -rf /var/etcd 
`
	RemoteRemoveAPIServerEtcHost = "sed -i \"/%s/d\" /etc/hosts"
	RemoveLvscareStaticPod       = "rm -rf  /etc/kubernetes/manifests/kube-sealyun-lvscare*"
	CreateLvscareStaticPod       = "mkdir -p /etc/kubernetes/manifests && echo '%s' > /etc/kubernetes/manifests/kube-sealyun-lvscare.yaml"
	KubeDeleteNode               = "kubectl delete node %s"
	// TODO check kubernetes certs
	RemoteCheckCerts = "kubeadm alpha certs check-expiration"
)

const (
	AdminConf      = "admin.conf"
	ControllerConf = "controller-manager.conf"
	SchedulerConf  = "scheduler.conf"
	KubeletConf    = "kubelet.conf"

	// kube file
	KUBECONTROLLERCONFIGFILE = "/etc/kubernetes/controller-manager.conf"
	KUBESCHEDULERCONFIGFILE  = "/etc/kubernetes/scheduler.conf"

	// CriSocket
	DefaultDockerCRISocket     = "/var/run/dockershim.sock"
	DefaultContainerdCRISocket = "/run/containerd/containerd.sock"
	DefaultSystemdCgroupDriver = "systemd"
	DefaultCgroupDriver        = "cgroupfs"

	// kubeadm api version
	KubeadmV1beta1 = "kubeadm.k8s.io/v1beta1"
	KubeadmV1beta2 = "kubeadm.k8s.io/v1beta2"
	KubeadmV1beta3 = "kubeadm.k8s.io/v1beta3"
)

const (
	Master0              = "Master0"
	Master               = "Master"
	Masters              = "Masters"
	TokenDiscovery       = "TokenDiscovery"
	VIP                  = "VIP"
	Version              = "Version"
	APIServer            = "ApiServer"
	PodCIDR              = "PodCIDR"
	SvcCIDR              = "SvcCIDR"
	Repo                 = "Repo"
	CertSANS             = "CertSANS"
	EtcdServers          = "etcd-servers"
	CriSocket            = "CriSocket"
	CriCGroupDriver      = "CriCGroupDriver"
	KubeadmAPI           = "KubeadmAPI"
	TokenDiscoveryCAHash = "TokenDiscoveryCAHash"
)

type CommandType string

//command type
const InitMaster CommandType = "initMaster"
const JoinMaster CommandType = "joinMaster"
const JoinNode CommandType = "joinNode"

func getAPIServerHost(ipAddr, APIServer string) (host string) {
	return fmt.Sprintf("%s %s", ipAddr, APIServer)
}

func (k *KubeadmRuntime) JoinMasterCommands(master, joinCmd, hostname string) []string {
	registryHost := getRegistryHost(k.getRootfs(), k.getMaster0IP())
	apiServerHost := getAPIServerHost(k.getMaster0IP(), k.getAPIServerDomain())
	cmdAddRegistryHosts := fmt.Sprintf(RemoteAddEtcHosts, registryHost, registryHost)
	certCMD := command.RemoteCerts(k.getCertSANS(), master, hostname, k.getSvcCIDR(), "")
	cmdAddHosts := fmt.Sprintf(RemoteAddEtcHosts, apiServerHost, apiServerHost)
	joinCommands := []string{cmdAddRegistryHosts, certCMD, cmdAddHosts}
	cf := GetRegistryConfig(k.getImageMountDir(), k.getMaster0IP())
	if cf.Username != "" && cf.Password != "" {
		joinCommands = append(joinCommands, fmt.Sprintf(DockerLoginCommand, cf.Domain+":"+cf.Port, cf.Username, cf.Password))
	}
	cmdUpdateHosts := fmt.Sprintf(RemoteUpdateEtcHosts, apiServerHost,
		getAPIServerHost(master, k.getAPIServerDomain()))

	return append(joinCommands, joinCmd, cmdUpdateHosts, RemoteCopyKubeConfig)
}

func (k *KubeadmRuntime) sendKubeConfigFile(hosts []string, kubeFile string) error {
	absKubeFile := fmt.Sprintf("%s/%s", cert.KubernetesDir, kubeFile)
	sealerKubeFile := fmt.Sprintf("%s/%s", k.getBasePath(), kubeFile)
	return k.sendFileToHosts(hosts, sealerKubeFile, absKubeFile)
}

func (k *KubeadmRuntime) sendNewCertAndKey(hosts []string) error {
	err := k.sendFileToHosts(hosts, k.getPKIPath(), cert.KubeDefaultCertPath)
	if err != nil {
		return err
	}
	return k.sendFileToHosts(k.getMasterIPList()[:1], k.getCertsDir(), filepath.Join(k.getRootfs(), "certs"))
}

func (k *KubeadmRuntime) sendRegistryCert(host []string) error {
	err := k.sendFileToHosts(host, fmt.Sprintf("%s/%s.crt", k.getCertsDir(), SeaHub), fmt.Sprintf("%s/%s/%s.crt", DockerCertDir, SeaHub, SeaHub))
	if err != nil {
		return err
	}
	return k.sendFileToHosts(host, fmt.Sprintf("%s/%s.crt", k.getCertsDir(), SeaHub), fmt.Sprintf("%s/%s:%d/%s.crt", DockerCertDir, SeaHub, k.getDefaultRegistryPort(), SeaHub))
}

func (k *KubeadmRuntime) sendFileToHosts(Hosts []string, src, dst string) error {
	g, _ := errgroup.WithContext(context.Background())
	for _, node := range Hosts {
		node := node
		g.Go(func() error {
			ssh, err := k.getHostSSHClient(node)
			if err != nil {
				return fmt.Errorf("send file failed %v", err)
			}
			if err := ssh.Copy(node, src, dst); err != nil {
				return fmt.Errorf("send file failed %v", err)
			}
			return err
		})
	}
	return g.Wait()
}

func (k *KubeadmRuntime) ReplaceKubeConfigV1991V1992(masters []string) bool {
	// fix > 1.19.1 kube-controller-manager and kube-scheduler use the LocalAPIEndpoint instead of the ControlPlaneEndpoint.
	if k.getKubeVersion() == V1991 || k.getKubeVersion() == V1992 {
		for _, v := range masters {
			cmd := fmt.Sprintf(RemoteReplaceKubeConfig, KUBESCHEDULERCONFIGFILE, v, KUBECONTROLLERCONFIGFILE, v, KUBESCHEDULERCONFIGFILE)
			ssh, err := k.getHostSSHClient(v)
			if err != nil {
				logger.Info("failed to replace kube config on %s:%v ", v, err)
				return false
			}
			if err := ssh.CmdAsync(v, cmd); err != nil {
				logger.Info("failed to replace kube config on %s:%v ", v, err)
				return false
			}
		}
		return true
	}
	return false
}

func (k *KubeadmRuntime) SendJoinMasterKubeConfigs(masters []string, files ...string) error {
	for _, f := range files {
		if err := k.sendKubeConfigFile(masters, f); err != nil {
			return err
		}
	}
	if k.ReplaceKubeConfigV1991V1992(masters) {
		logger.Info("set kubernetes v1.19.1 v1.19.2 kube config")
	}
	return nil
}

// JoinTemplate is generate JoinCP nodes configuration by master ip.
func (k *KubeadmRuntime) joinMasterConfig(masterIP string) ([]byte, error) {
	k.Lock()
	defer k.Unlock()
	// TODO Using join file instead template
	k.setAPIServerEndpoint(fmt.Sprintf("%s:6443", k.getMaster0IP()))
	k.setJoinAdvertiseAddress(masterIP)
	k.setCgroupDriver(k.getCgroupDriverFromShell(masterIP))
	return utils.MarshalConfigsYaml(k.JoinConfiguration, k.KubeletConfiguration)
}

// sendJoinCPConfig send join CP nodes configuration
func (k *KubeadmRuntime) sendJoinCPConfig(joinMaster []string) error {
	k.Mutex = &sync.Mutex{}
	g, _ := errgroup.WithContext(context.Background())
	for _, master := range joinMaster {
		master := master
		g.Go(func() error {
			joinConfig, err := k.joinMasterConfig(master)
			if err != nil {
				return fmt.Errorf("get join %s config failed: %v", master, err)
			}
			cmd := fmt.Sprintf(RemoteJoinMasterConfig, joinConfig, k.getRootfs())
			ssh, err := k.getHostSSHClient(master)
			if err != nil {
				return fmt.Errorf("set join kubeadm config failed %s %s %v", master, cmd, err)
			}
			if err := ssh.CmdAsync(master, cmd); err != nil {
				return fmt.Errorf("set join kubeadm config failed %s %s %v", master, cmd, err)
			}
			return err
		})
	}
	return g.Wait()
}

func (k *KubeadmRuntime) CmdAsyncHosts(hosts []string, cmd string) error {
	g, _ := errgroup.WithContext(context.Background())
	for _, host := range hosts {
		host := host
		g.Go(func() error {
			ssh, err := k.getHostSSHClient(host)
			if err != nil {
				logger.Error("exec command failed %s %s %v", host, cmd, err)
			}
			if err := ssh.CmdAsync(host, cmd); err != nil {
				logger.Error("exec command failed %s %s %v", host, cmd, err)
			}
			return err
		})
	}
	return g.Wait()
}

func vlogToStr(vlog int) string {
	str := strconv.Itoa(vlog)
	return " -v " + str
}

func (k *KubeadmRuntime) Command(version string, name CommandType) (cmd string) {
	//cmds := make(map[CommandType]string)
	// Please convert your v1beta1 configuration files to v1beta2 using the
	// "kubeadm config migrate" command of kubeadm v1.15.x, so v1.14 not support multi network interface.
	cmds := map[CommandType]string{
		InitMaster: fmt.Sprintf(InitMaster115Lower, k.getRootfs()),
		JoinMaster: fmt.Sprintf(JoinMaster115Lower, k.getMaster0IP(), k.getJoinToken(), k.getTokenCaCertHash(), k.getCertificateKey()),
		JoinNode:   fmt.Sprintf(JoinNode115Lower, k.getVIP(), k.getJoinToken(), k.getTokenCaCertHash()),
	}
	//other version >= 1.15.x
	if VersionCompare(version, V1150) {
		cmds[InitMaster] = fmt.Sprintf(InitMaser115Upper, k.getRootfs())
		cmds[JoinMaster] = fmt.Sprintf(JoinMaster115Upper, k.getRootfs())
		cmds[JoinNode] = fmt.Sprintf(JoinNode115Upper, k.getRootfs())
	}

	v, ok := cmds[name]
	if !ok {
		logger.Error("get kubeadm command failed %v", cmds)
		return ""
	}

	if utils.IsInContainer() {
		return fmt.Sprintf("%s%s%s", v, vlogToStr(k.Vlog), " --ignore-preflight-errors=all")
	}
	if name == InitMaster || name == JoinMaster {
		return fmt.Sprintf("%s%s%s", v, vlogToStr(k.Vlog), " --ignore-preflight-errors=SystemVerification")
	}

	return fmt.Sprintf("%s%s", v, vlogToStr(k.Vlog))
}

func (k *KubeadmRuntime) GetRemoteHostName(hostIP string) string {
	hostName := k.CmdToString(hostIP, "hostname", "")
	return strings.ToLower(hostName)
}

func (k *KubeadmRuntime) joinMasters(masters []string) error {
	if len(masters) == 0 {
		return nil
	}
	// if its do not Load and Merge kubeadm config via init, need to redo it
	if err := k.MergeKubeadmConfig(); err != nil {
		return err
	}
	if err := k.WaitSSHReady(6, masters...); err != nil {
		return errors.Wrap(err, "join masters wait for ssh ready time out")
	}
	if err := k.GetJoinTokenHashAndKey(); err != nil {
		return err
	}
	if err := k.CopyStaticFiles(masters); err != nil {
		return err
	}
	if err := k.SendJoinMasterKubeConfigs(masters, AdminConf, ControllerConf, SchedulerConf); err != nil {
		return err
	}
	if err := k.sendRegistryCert(masters); err != nil {
		return err
	}
	// TODO only needs send ca?
	if err := k.sendNewCertAndKey(masters); err != nil {
		return err
	}
	if err := k.sendJoinCPConfig(masters); err != nil {
		return err
	}
	cmd := k.Command(k.getKubeVersion(), JoinMaster)
	// TODO for test skip dockerd dev version
	if cmd == "" {
		return fmt.Errorf("get join master command failed, kubernetes version is %s", k.getKubeVersion())
	}

	for _, master := range masters {
		logger.Info("Start to join %s as master", master)

		hostname := k.GetRemoteHostName(master)
		if hostname == "" {
			return fmt.Errorf("get remote hostname failed %s", master)
		}
		cmds := k.JoinMasterCommands(master, cmd, hostname)
		ssh, err := k.getHostSSHClient(master)
		if err != nil {
			return err
		}

		if err := ssh.CmdAsync(master, cmds...); err != nil {
			return fmt.Errorf("exec command failed %s %v %v", master, cmds, err)
		}

		logger.Info("Succeeded in joining %s as master", master)
	}
	return nil
}

func (k *KubeadmRuntime) deleteMasters(masters []string) error {
	if len(masters) == 0 {
		return nil
	}
	g, _ := errgroup.WithContext(context.Background())
	for _, master := range masters {
		master := master
		g.Go(func() error {
			master := master
			logger.Info("Start to delete master %s", master)
			if err := k.deleteMaster(master); err != nil {
				logger.Error("delete master %s failed %v", master, err)
			}
			logger.Info("Succeeded in deleting master %s", master)
			return nil
		})
	}
	return g.Wait()
}

func SliceRemoveStr(ss []string, s string) (result []string) {
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return
}

func (k *KubeadmRuntime) isHostName(master, host string) string {
	hostString := k.CmdToString(master, "kubectl get nodes | grep -v NAME  | awk '{print $1}'", ",")
	hostName := k.CmdToString(host, "hostname", "")
	hosts := strings.Split(hostString, ",")
	var name string
	for _, h := range hosts {
		if strings.TrimSpace(h) == "" {
			continue
		} else {
			hh := strings.ToLower(h)
			fromH := strings.ToLower(hostName)
			if hh == fromH {
				name = h
				break
			}
		}
	}
	return name
}

func (k *KubeadmRuntime) deleteMaster(master string) error {
	ssh, err := k.getHostSSHClient(master)
	if err != nil {
		return fmt.Errorf("failed to delete master: %v", err)
	}
	remoteCleanCmd := []string{fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(k.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(k.getRootfs(), k.getMaster0IP())),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, k.getAPIServerDomain())}

	//if the master to be removed is the execution machine, kubelet and ~./kube will not be removed and ApiServer host will be added.
	address, err := utils.GetLocalHostAddresses()
	if err != nil || !utils.IsLocalIP(master, address) {
		remoteCleanCmd = append(remoteCleanCmd, RemoveKubeConfig)
	} else {
		apiServerHost := getAPIServerHost(k.getMaster0IP(), k.getAPIServerDomain())
		remoteCleanCmd = append(remoteCleanCmd,
			fmt.Sprintf(RemoteAddEtcHosts, apiServerHost, apiServerHost))
	}
	if err := ssh.CmdAsync(master, remoteCleanCmd...); err != nil {
		return err
	}

	//remove master
	masterIPs := SliceRemoveStr(k.getMasterIPList(), master)
	if len(masterIPs) > 0 {
		hostname := k.isHostName(k.getMaster0IP(), master)
		master0SSH, err := k.getHostSSHClient(k.getMaster0IP())
		if err != nil {
			return fmt.Errorf("failed to remove master ip: %v", err)
		}

		if err := master0SSH.CmdAsync(k.getMaster0IP(), fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("delete node %s failed %v", hostname, err)
		}
	}
	yaml := ipvs.LvsStaticPodYaml(k.getVIP(), masterIPs, "")
	g, _ := errgroup.WithContext(context.Background())
	for _, node := range k.getNodesIPList() {
		node := node
		g.Go(func() error {
			ssh, err := k.getHostSSHClient(node)
			if err != nil {
				logger.Error("update lvscare static pod failed %s %v", node, err)
			}
			if err := ssh.CmdAsync(node, RemoveLvscareStaticPod, fmt.Sprintf(CreateLvscareStaticPod, yaml)); err != nil {
				logger.Error("update lvscare static pod failed %s %v", node, err)
			}
			return err
		})
	}
	return g.Wait()
}

func (k *KubeadmRuntime) GetJoinTokenHashAndKey() error {
	cmd := fmt.Sprintf(`kubeadm init phase upload-certs --upload-certs -v %d`, k.Vlog)
	/*
		I0415 11:45:06.653868   14520 version.go:251] remote version is much newer: v1.21.0; falling back to: stable-1.16
		[upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
		[upload-certs] Using certificate key:
		8376c70aaaf285b764b3c1a588740728aff493d7c2239684e84a7367c6a437cf
	*/
	output := k.CmdToString(k.getMaster0IP(), cmd, "\r\n")
	logger.Debug("[globals]decodeCertCmd: %s", output)
	slice := strings.Split(output, "Using certificate key:")
	if len(slice) != 2 {
		return fmt.Errorf("get certifacate key failed %s", slice)
	}
	key := strings.Replace(slice[1], "\r\n", "", -1)
	k.CertificateKey = strings.Replace(key, "\n", "", -1)
	cmd = fmt.Sprintf("kubeadm token create --print-join-command -v %d", k.Vlog)

	ssh, err := k.getHostSSHClient(k.getMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get join token hash and key: %v", err)
	}
	out, err := ssh.Cmd(k.getMaster0IP(), cmd)
	if err != nil {
		return fmt.Errorf("create kubeadm join token failed %v", err)
	}

	k.decodeMaster0Output(out)

	logger.Info("join token: %s hash: %s certifacate key: %s", k.getJoinToken(), k.getTokenCaCertHash(), k.getCertificateKey())
	return nil
}
