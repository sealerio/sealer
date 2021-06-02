package runtime

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils/ssh"

	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/command"
	"github.com/alibaba/sealer/ipvs"
	"github.com/alibaba/sealer/utils"
)

const (
	V1991 = "v1.19.1"
	V1992 = "v1.19.2"
	V1150 = "v1.15.0"
	V1200 = "v1.20.0"
)

const (
	RemoteRestartDocker     = "systemctl restart docker"
	RemoteAddEtcHosts       = "echo %s >> /etc/hosts"
	RemoteUpdateEtcHosts    = `sed "s/%s/%s/g" -i /etc/hosts`
	RemoteCopyKubeConfig    = `rm -rf .kube/config && mkdir -p /root/.kube && cp /etc/kubernetes/admin.conf /root/.kube/config`
	RemoteReplaceKubeConfig = `grep -qF "apiserver.cluster.local" %s  && sed -i 's/apiserver.cluster.local/%s/' %s && sed -i 's/apiserver.cluster.local/%s/' %s`
	RemoteJoinMasterConfig  = `echo "%s" > %s/kubeadm-join-config.yaml`
	InitMaster115Lower      = `kubeadm init --config=%s/kubeadm-config.yaml --experimental-upload-certs`
	JoinMaster115Lower      = "kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s --experimental-control-plane --certificate-key %s"
	JoinNode115Lower        = "kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s"
	InitMaser115Upper       = `kubeadm init --config=%s/kubeadm-config.yaml --upload-certs`
	JoinMaster115Upper      = "kubeadm join --config=%s/kubeadm-join-config.yaml"
	JoinNode115Upper        = "kubeadm join --config=%s/kubeadm-join-config.yaml"
	RemoteCleanMasterOrNode = `kubeadm reset -f %s && \
modprobe -r ipip  && lsmod && \
rm -rf ~/.kube/ && rm -rf /etc/kubernetes/ && \
rm -rf /etc/systemd/system/kubelet.service.d && rm -rf /etc/systemd/system/kubelet.service && \
rm -rf /usr/bin/kube* && rm -rf /usr/bin/crictl && \
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
	EtcdServers          = "EtcdServers"
	CriSocket            = "CriSocket"
	TokenDiscoveryCAHash = "TokenDiscoveryCAHash"
	SeaHub               = "sea.hub"
)

type CommandType string

//command type
const InitMaster CommandType = "initMaster"
const JoinMaster CommandType = "joinMaster"
const JoinNode CommandType = "joinNode"

func getAPIServerHost(ipAddr, APIServer string) (host string) {
	return fmt.Sprintf("%s %s", ipAddr, APIServer)
}

func (d *Default) JoinMasterCommands(master, joinCmd, hostname string) []string {
	cmdAddRegistryHosts := fmt.Sprintf(RemoteAddEtcHosts, getRegistryHost(utils.GetHostIP(d.Masters[0])))
	hostIP := utils.GetHostIP(master)
	certCMD := command.RemoteCerts(d.APIServerCertSANs, hostIP, hostname, d.SvcCIDR, "")
	cmdAddHosts := fmt.Sprintf(RemoteAddEtcHosts, getAPIServerHost(utils.GetHostIP(d.Masters[0]), d.APIServer))
	cmdUpdateHosts := fmt.Sprintf(RemoteUpdateEtcHosts, getAPIServerHost(utils.GetHostIP(d.Masters[0]), d.APIServer),
		getAPIServerHost(utils.GetHostIP(master), d.APIServer))

	return []string{cmdAddRegistryHosts, certCMD, cmdAddHosts, joinCmd, cmdUpdateHosts, RemoteCopyKubeConfig}
}

func (d *Default) sendKubeConfigFile(hosts []string, kubeFile string) {
	absKubeFile := fmt.Sprintf("%s/%s", cert.KubernetesDir, kubeFile)
	sealerKubeFile := fmt.Sprintf("%s/%s", d.BasePath, kubeFile)
	d.sendFileToHosts(hosts, sealerKubeFile, absKubeFile)
}

func (d *Default) sendNewCertAndKey(hosts []string) {
	d.sendFileToHosts(hosts, d.CertPath, cert.KubeDefaultCertPath)
}

func (d *Default) sendFileToHosts(Hosts []string, src, dst string) {
	var wg sync.WaitGroup
	for _, node := range Hosts {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			err := d.SSH.Copy(node, src, dst)
			if err != nil {
				logger.Error("send file failed %v", err)
			}
		}(node)
	}
	wg.Wait()
}

func (d *Default) ReplaceKubeConfigV1991V1992(masters []string) bool {
	// fix > 1.19.1 kube-controller-manager and kube-scheduler use the LocalAPIEndpoint instead of the ControlPlaneEndpoint.
	if d.Metadata.Version == V1991 || d.Metadata.Version == V1992 {
		for _, v := range masters {
			ip := utils.GetHostIP(v)
			cmd := fmt.Sprintf(RemoteReplaceKubeConfig, KUBESCHEDULERCONFIGFILE, ip, KUBECONTROLLERCONFIGFILE, ip, KUBESCHEDULERCONFIGFILE)
			err := d.SSH.CmdAsync(v, cmd)
			if err != nil {
				logger.Info("failed to replace kube config on %s ", v)
				return false
			}
		}
		return true
	}
	return false
}

func (d *Default) SendJoinMasterKubeConfigs(masters []string, files ...string) {
	for _, f := range files {
		d.sendKubeConfigFile(masters, f)
	}
	if d.ReplaceKubeConfigV1991V1992(masters) {
		logger.Info("set kubernetes v1.19.1 v1.19.2 kube config")
	}
}

func joinKubeadmConfig() string {
	var sb strings.Builder
	sb.Write([]byte(JoinCPTemplateTextV1beta2))
	return sb.String()
}

func (d *Default) JoinTemplateFromTemplateContent(templateContent, ip string) []byte {
	tmpl, err := template.New("text").Parse(templateContent)
	if err != nil {
		logger.Error("template join config failed %v", err)
		return []byte{}
	}
	var envMap = make(map[string]interface{})
	envMap[Master0] = utils.GetHostIP(d.Masters[0])
	envMap[Master] = ip
	envMap[TokenDiscovery] = d.JoinToken
	envMap[TokenDiscoveryCAHash] = d.TokenCaCertHash
	envMap[VIP] = d.VIP
	if VersionCompare(d.Metadata.Version, V1200) {
		envMap[CriSocket] = DefaultContainerdCRISocket
	} else {
		envMap[CriSocket] = DefaultDockerCRISocket
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, envMap)
	if err != nil {
		logger.Error("render join template failed %v", err)
	}
	return buffer.Bytes()
}

// JoinTemplate is generate JoinCP nodes configuration by master ip.
func (d *Default) JoinTemplate(ip string) []byte {
	return d.JoinTemplateFromTemplateContent(joinKubeadmConfig(), ip)
}

// sendJoinCPConfig send join CP nodes configuration
func (d *Default) sendJoinCPConfig(joinMaster []string) {
	var wg sync.WaitGroup
	for _, master := range joinMaster {
		wg.Add(1)
		go func(master string) {
			defer wg.Done()
			templateData := string(d.JoinTemplate(utils.GetHostIP(master)))
			cmd := fmt.Sprintf(RemoteJoinMasterConfig, templateData, d.Rootfs)
			err := d.SSH.CmdAsync(master, cmd)
			if err != nil {
				logger.Error("set join kubeadm config failed %s %s %v", master, cmd, err)
			}
		}(master)
	}
	wg.Wait()
}

func (d *Default) CmdAsyncHosts(hosts []string, cmd string) error {
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			err := d.SSH.CmdAsync(host, cmd)
			if err != nil {
				logger.Error("exec command failed %s %s %v", host, cmd, err)
			}
		}(host)
	}
	wg.Wait()
	return nil
}

func vlogToStr(vlog int) string {
	str := strconv.Itoa(vlog)
	return " -v " + str
}

func (d *Default) Command(version string, name CommandType) (cmd string) {
	//cmds := make(map[CommandType]string)
	// Please convert your v1beta1 configuration files to v1beta2 using the
	// "kubeadm config migrate" command of kubeadm v1.15.x, so v1.14 not support multi network interface.
	cmds := map[CommandType]string{
		InitMaster: fmt.Sprintf(InitMaster115Lower, d.Rootfs),
		JoinMaster: fmt.Sprintf(JoinMaster115Lower, utils.GetHostIP(d.Masters[0]), d.JoinToken, d.TokenCaCertHash, d.CertificateKey),
		JoinNode:   fmt.Sprintf(JoinNode115Lower, d.VIP, d.JoinToken, d.TokenCaCertHash),
	}
	//other version >= 1.15.x
	if VersionCompare(version, V1150) {
		cmds[InitMaster] = fmt.Sprintf(InitMaser115Upper, d.Rootfs)
		cmds[JoinMaster] = fmt.Sprintf(JoinMaster115Upper, d.Rootfs)
		cmds[JoinNode] = fmt.Sprintf(JoinNode115Upper, d.Rootfs)
	}

	v, ok := cmds[name]
	if !ok {
		logger.Error("get kubeadm command failed %v", cmds)
		return ""
	}
	return fmt.Sprintf("%s%s", v, vlogToStr(d.Vlog))
}

//CmdToString is in host exec cmd and replace to spilt str
func (d *Default) CmdToString(host, cmd, split string) string {
	data, err := d.SSH.Cmd(host, cmd)
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

func (d *Default) GetRemoteHostName(hostIP string) string {
	hostName := d.CmdToString(hostIP, "hostname", "")
	return strings.ToLower(hostName)
}

func (d *Default) joinMasters(masters []string) error {
	if len(masters) == 0 {
		return nil
	}
	if err := d.LoadMetadata(); err != nil {
		return fmt.Errorf("failed to load metadata %v", err)
	}
	if err := ssh.WaitSSHReady(d.SSH, masters...); err != nil {
		return errors.Wrap(err, "join masters wait for ssh ready time out")
	}
	if err := d.GetJoinTokenHashAndKey(); err != nil {
		return err
	}
	if err := d.CopyStaticFiles(masters); err != nil {
		return err
	}
	d.SendJoinMasterKubeConfigs(masters, AdminConf, ControllerConf, SchedulerConf)
	// TODO only needs send ca?
	d.sendNewCertAndKey(masters)
	d.sendJoinCPConfig(masters)
	cmd := d.Command(d.Metadata.Version, JoinMaster)
	// TODO for test skip dockerd dev version
	cmd = fmt.Sprintf("%s --ignore-preflight-errors=SystemVerification", cmd)
	if cmd == "" {
		return fmt.Errorf("get join master command failed, kubernetes version is %s", d.Metadata.Version)
	}

	for _, master := range masters {
		hostname := d.GetRemoteHostName(master)
		if hostname == "" {
			return fmt.Errorf("get remote hostname failed %s", master)
		}
		cmds := d.JoinMasterCommands(master, cmd, hostname)
		if err := d.SSH.CmdAsync(master, cmds...); err != nil {
			return fmt.Errorf("exec command failed %s %v %v", master, cmds, err)
		}
	}
	return nil
}

/*func (d *Default) joinMastersAsync(masters []string) error {
	d.SendJoinMasterKubeConfigs(masters)
	d.sendNewCertAndKey(masters)
	d.sendJoinCPConfig(masters)
	cmd := d.Command(d.Metadata.Version, JoinMaster)
	if cmd == "" {
		return fmt.Errorf("get join master command failed, kubernetes version is %s", d.Metadata.Version)
	}

	var wg sync.WaitGroup
	for _, master := range masters {
		hostname := d.GetRemoteHostName(master)
		if hostname == "" {
			return fmt.Errorf("get remote hostname failed %s", master)
		}
		wg.Add(1)
		go func(master, hostname string) {
			defer wg.Done()
			cmds := d.JoinMasterCommands(master, cmd, hostname)
			if err := d.SSH.CmdAsync(master, cmds...); err != nil {
				logger.Error("exec command failed %s %v %v", master, cmds, err)
				return
			}
		}(master, hostname)
	}
	wg.Wait()
	return nil
}*/

func (d *Default) deleteMasters(masters []string) error {
	if len(masters) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, master := range masters {
		wg.Add(1)
		go func(master string) {
			defer wg.Done()
			if err := d.deleteMaster(master); err != nil {
				logger.Error("delete master %s failed %v", master, err)
			}
		}(master)
	}
	wg.Wait()

	return nil
}

func SliceRemoveStr(ss []string, s string) (result []string) {
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return
}

func (d *Default) isHostName(master, host string) string {
	hostString := d.CmdToString(master, "kubectl get nodes | grep -v NAME  | awk '{print $1}'", ",")
	hostName := d.CmdToString(host, "hostname", "")
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

func (d *Default) deleteMaster(master string) error {
	host := utils.GetHostIP(master)
	if err := d.SSH.CmdAsync(host, fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(d.Vlog)), fmt.Sprintf(RemoteRemoveAPIServerEtcHost, d.APIServer), fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(d.Masters[0]))); err != nil {
		return err
	}

	//remove master
	masterIPs := SliceRemoveStr(d.Masters, master)
	if len(masterIPs) > 0 {
		hostname := d.isHostName(masterIPs[0], master)
		err := d.SSH.CmdAsync(masterIPs[0], fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname)))
		if err != nil {
			return fmt.Errorf("delete node %s failed %v", hostname, err)
		}
	}
	yaml := ipvs.LvsStaticPodYaml(d.VIP, masterIPs, d.LvscareImage)
	var wg sync.WaitGroup
	for _, node := range d.Nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			if err := d.SSH.CmdAsync(node, RemoveLvscareStaticPod, fmt.Sprintf(CreateLvscareStaticPod, yaml)); err != nil {
				logger.Error("update lvscare static pod failed %s %v", node, err)
			}
		}(node)
	}
	wg.Wait()

	return nil
}

func (d *Default) GetJoinTokenHashAndKey() error {
	cmd := fmt.Sprintf(`kubeadm init phase upload-certs --upload-certs -v %d`, d.Vlog)
	/*
		I0415 11:45:06.653868   14520 version.go:251] remote version is much newer: v1.21.0; falling back to: stable-1.16
		[upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
		[upload-certs] Using certificate key:
		8376c70aaaf285b764b3c1a588740728aff493d7c2239684e84a7367c6a437cf
	*/
	output := d.CmdToString(d.Masters[0], cmd, "\r\n")
	logger.Debug("[globals]decodeCertCmd: %s", output)
	slice := strings.Split(output, "Using certificate key:")
	if len(slice) != 2 {
		return fmt.Errorf("get certifacate key failed %s", slice)
	}
	key := strings.Replace(slice[1], "\r\n", "", -1)
	d.CertificateKey = strings.Replace(key, "\n", "", -1)
	cmd = fmt.Sprintf("kubeadm token create --print-join-command -v %d", d.Vlog)
	out, err := d.SSH.Cmd(d.Masters[0], cmd)
	if err != nil {
		return fmt.Errorf("create kubeadm join token failed %v", err)
	}

	d.decodeMaster0Output(out)

	logger.Info("join token: %s hash: %s certifacate key: %s", d.Token, d.TokenCaCertHash, d.CertificateKey)
	return nil
}
