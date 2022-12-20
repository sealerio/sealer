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

package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containers/common/pkg/auth"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
	osutils "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/shellcommand"
)

type localConfigurator struct {
	*v2.LocalRegistry
	deployHosts          []net.IP
	containerRuntimeInfo containerruntime.Info
	infraDriver          infradriver.InfraDriver
	distributor          imagedistributor.Distributor
}

func (c *localConfigurator) GetDriver() (Driver, error) {
	endpoint := net.JoinHostPort(c.Domain, strconv.Itoa(c.Port))
	dataDir := filepath.Join(c.infraDriver.GetClusterRootfsPath(), "registry")
	return newLocalRegistryDriver(endpoint, dataDir, c.deployHosts, c.distributor), nil
}

func (c *localConfigurator) UninstallFrom(deletedMasters, deletedNodes []net.IP) error {
	// remove all registry configs on /etc/hosts
	all := append(deletedMasters, deletedNodes...)
	if err := c.removeRegistryConfig(all); err != nil {
		return err
	}
	if !*c.HA {
		return nil
	}
	// if current deployHosts is null,means clean all, just return.
	if len(c.deployHosts) == 0 {
		return nil
	}

	// flush ipvs policy on remain nodes
	var rs []string
	for _, m := range c.deployHosts {
		rs = append(rs, fmt.Sprintf("--rs %s", net.JoinHostPort(m.String(), strconv.Itoa(c.Port))))
	}

	vs := net.JoinHostPort(common.DefaultVIP, strconv.Itoa(c.Port))
	ipvsCmd := fmt.Sprintf("seautil ipvs --vs %s %s --health-path /healthz --health-schem https --run-once", vs, strings.Join(rs, " "))
	remainNodes := utilsnet.RemoveIPs(c.infraDriver.GetHostIPListByRole(common.NODE), deletedNodes)
	for _, node := range remainNodes {
		if err := c.infraDriver.CmdAsync(node, ipvsCmd); err != nil {
			return fmt.Errorf("flush ipvs policy on remain nodes(%s): %v", node.String(), err)
		}
	}

	return nil
}

func (c *localConfigurator) removeRegistryConfig(hosts []net.IP) error {
	var uninstallCmd []string
	if c.RegistryConfig.Username != "" && c.RegistryConfig.Password != "" {
		//todo use sdk to logout instead of shell cmd
		logoutCmd := fmt.Sprintf("docker logout %s ", net.JoinHostPort(c.Domain, strconv.Itoa(c.Port)))
		if c.containerRuntimeInfo.Type != "docker" {
			logoutCmd = fmt.Sprintf("nerdctl logout %s ", net.JoinHostPort(c.Domain, strconv.Itoa(c.Port)))
		}
		uninstallCmd = append(uninstallCmd, logoutCmd)
	}

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, strings.Join(uninstallCmd, "&&"))
		if err != nil {
			return fmt.Errorf("failed to delete registry configuration: %v", err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localConfigurator) InstallOn(masters, nodes []net.IP) error {
	hosts := append(masters, nodes...)
	logrus.Infof("will install local private registry configuration on %+v\n", hosts)
	if err := c.configureRegistryNetwork(masters, nodes); err != nil {
		return err
	}

	if err := c.configureRegistryCert(hosts); err != nil {
		return err
	}

	if err := c.configureDaemonService(hosts); err != nil {
		return err
	}

	if err := c.configureAccessCredential(hosts); err != nil {
		return err
	}

	return nil
}

// add registry domain and ip to "/etc/hosts"
// add registry ip to ipvs policy
func (c *localConfigurator) configureRegistryNetwork(masters, nodes []net.IP) error {
	if !*c.HA {
		return c.configureSingletonHostsFile(append(masters, nodes...))
	}

	// for master: domain + local IP
	for _, m := range masters {
		cmd := shellcommand.CommandSetHostAlias(c.Domain, m.String())
		if err := c.infraDriver.CmdAsync(m, cmd); err != nil {
			return fmt.Errorf("failed to config masters hosts file: %v", err)
		}
	}

	// for node: add ipvs policy; domain + VIP
	var rs []string
	for _, m := range c.deployHosts {
		rs = append(rs, fmt.Sprintf("--rs %s", net.JoinHostPort(m.String(), strconv.Itoa(c.Port))))
	}

	vs := net.JoinHostPort(common.DefaultVIP, strconv.Itoa(c.Port))
	ipvsCmd := fmt.Sprintf("seautil ipvs --vs %s %s --health-path /healthz --health-schem https --run-once", vs, strings.Join(rs, " "))
	// flush all cluster nodes as latest ipvs policy.
	currentNodes := c.infraDriver.GetHostIPListByRole(common.NODE)
	for _, n := range currentNodes {
		err := c.infraDriver.CmdAsync(n, ipvsCmd)
		if err != nil {
			return fmt.Errorf("failed to config ndoes lvs policy %s: %v", ipvsCmd, err)
		}

		err = c.infraDriver.CmdAsync(n, shellcommand.CommandSetHostAlias(c.Domain, common.DefaultVIP))
		if err != nil {
			return fmt.Errorf("failed to config ndoes hosts file cmd: %v", err)
		}
	}
	return nil
}

func (c *localConfigurator) configureSingletonHostsFile(hosts []net.IP) error {
	// add registry ip to "/etc/hosts"
	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, shellcommand.CommandSetHostAlias(c.Domain, c.deployHosts[0].String()))
		if err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localConfigurator) configureRegistryCert(hosts []net.IP) error {
	// if deploy registry as InsecureMode ,skip to configure cert.
	if *c.Insecure {
		return nil
	}

	var (
		endpoint = net.JoinHostPort(c.Domain, strconv.Itoa(c.Port))
		caFile   = c.Domain + ".crt"
		src      = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "certs", caFile)
		dest     = filepath.Join(c.containerRuntimeInfo.CertsDir, endpoint, caFile)
	)

	return c.copy2RemoteHosts(src, dest, hosts)
}

func (c *localConfigurator) configureAccessCredential(hosts []net.IP) error {
	var (
		username        = c.RegistryConfig.Username
		password        = c.RegistryConfig.Password
		endpoint        = net.JoinHostPort(c.Domain, strconv.Itoa(c.Port))
		tmpAuthFilePath = "/tmp/config.json"
		// todo we need this config file when kubelet pull images from registry. while, we could optimize the logic here.
		remoteKubeletAuthFilePath = "/var/lib/kubelet/config.json"
	)

	if username == "" || password == "" {
		return nil
	}

	err := auth.Login(context.TODO(),
		nil,
		&auth.LoginOptions{
			AuthFile:           tmpAuthFilePath,
			Password:           password,
			Username:           username,
			Stdout:             os.Stdout,
			AcceptRepositories: true,
		},
		[]string{endpoint})

	if err != nil {
		return err
	}

	defer func() {
		err = os.Remove(tmpAuthFilePath)
		if err != nil {
			logrus.Debugf("failed to remove tmp registry auth file:%s", tmpAuthFilePath)
		}
	}()

	err = c.copy2RemoteHosts(tmpAuthFilePath, c.containerRuntimeInfo.ConfigFilePath, hosts)
	if err != nil {
		return err
	}

	err = c.copy2RemoteHosts(tmpAuthFilePath, remoteKubeletAuthFilePath, hosts)
	if err != nil {
		return err
	}

	return nil
}

func (c *localConfigurator) copy2RemoteHosts(src, dest string, hosts []net.IP) error {
	f := func(host net.IP) error {
		err := c.infraDriver.Copy(host, src, dest)
		if err != nil {
			return fmt.Errorf("failed to copy local file %s to remote %s : %v", src, dest, err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localConfigurator) configureDaemonService(hosts []net.IP) error {
	var (
		src      string
		dest     string
		endpoint = net.JoinHostPort(c.Domain, strconv.Itoa(c.Port))
	)

	if endpoint == common.DefaultRegistryURL {
		return nil
	}

	if c.containerRuntimeInfo.Config.Type == "docker" {
		src = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "etc", "daemon.json")
		dest = "/etc/docker/daemon.json"
		if err := c.configureDockerDaemonService(endpoint, src); err != nil {
			return err
		}
	}

	if c.containerRuntimeInfo.Config.Type == "containerd" {
		src = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "etc", "hosts.toml")
		dest = filepath.Join("/etc/containerd/certs.d", endpoint, "hosts.toml")
		if err := c.configureContainerdDaemonService(endpoint, src); err != nil {
			return err
		}
	}

	// for docker: copy daemon.json to "/etc/docker/daemon.json"
	// for containerd : copy hosts.toml to "/etc/containerd/certs.d/${domain}:${port}/hosts.toml"
	for _, ip := range hosts {
		err := c.infraDriver.Copy(ip, src, dest)
		if err != nil {
			return err
		}

		err = c.infraDriver.CmdAsync(ip, "systemctl daemon-reload")
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *localConfigurator) configureDockerDaemonService(endpoint, daemonFile string) error {
	var daemonConf DaemonConfig

	b, err := os.ReadFile(filepath.Clean(daemonFile))
	if err != nil {
		return err
	}

	b = bytes.TrimSpace(b)
	// if config file is empty, only add registry config.
	if len(b) != 0 {
		if err := json.Unmarshal(b, &daemonConf); err != nil {
			return fmt.Errorf("failed to load %s to DaemonConfig: %v", daemonFile, err)
		}
	}

	daemonConf.RegistryMirrors = append(daemonConf.RegistryMirrors, "https://"+endpoint)

	content, err := json.MarshalIndent(daemonConf, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal daemonFile: %v", err)
	}

	return osutils.NewCommonWriter(daemonFile).WriteFile(content)
}

func (c *localConfigurator) configureContainerdDaemonService(endpoint, hostTomlFile string) error {
	var (
		caFile             = c.Domain + ".crt"
		registryCaCertPath = filepath.Join(c.containerRuntimeInfo.CertsDir, endpoint, caFile)
		url                = "https://" + endpoint
	)

	tree, err := toml.TreeFromMap(map[string]interface{}{
		"server": url,
		fmt.Sprintf(`host."%s"`, url): map[string]interface{}{
			"ca": registryCaCertPath,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal Containerd hosts.toml file: %v", err)
	}
	return osutils.NewCommonWriter(hostTomlFile).WriteFile([]byte(tree.String()))
}

type DaemonConfig struct {
	AllowNonDistributableArtifacts []string          `json:"allow-nondistributable-artifacts,omitempty"`
	APICorsHeader                  string            `json:"api-cors-header,omitempty"`
	AuthorizationPlugins           []string          `json:"authorization-plugins,omitempty"`
	Bip                            string            `json:"bip,omitempty"`
	Bridge                         string            `json:"bridge,omitempty"`
	CgroupParent                   string            `json:"cgroup-parent,omitempty"`
	ClusterAdvertise               string            `json:"cluster-advertise,omitempty"`
	ClusterStore                   string            `json:"cluster-store,omitempty"`
	Containerd                     string            `json:"containerd,omitempty"`
	ContainerdNamespace            string            `json:"containerd-namespace,omitempty"`
	ContainerdPluginNamespace      string            `json:"containerd-plugin-namespace,omitempty"`
	DataRoot                       string            `json:"data-root,omitempty"`
	Debug                          bool              `json:"debug,omitempty"`
	DefaultCgroupnsMode            string            `json:"default-cgroupns-mode,omitempty"`
	DefaultGateway                 string            `json:"default-gateway,omitempty"`
	DefaultGatewayV6               string            `json:"default-gateway-v6,omitempty"`
	DefaultRuntime                 string            `json:"default-runtime,omitempty"`
	DefaultShmSize                 string            `json:"default-shm-size,omitempty"`
	DNS                            []string          `json:"dns,omitempty"`
	DNSOpts                        []string          `json:"dns-opts,omitempty"`
	DNSSearch                      []string          `json:"dns-search,omitempty"`
	ExecOpts                       []string          `json:"exec-opts,omitempty"`
	ExecRoot                       string            `json:"exec-root,omitempty"`
	Experimental                   bool              `json:"experimental,omitempty"`
	FixedCidr                      string            `json:"fixed-cidr,omitempty"`
	FixedCidrV6                    string            `json:"fixed-cidr-v6,omitempty"`
	Group                          string            `json:"group,omitempty"`
	Hosts                          []string          `json:"hosts,omitempty"`
	Icc                            bool              `json:"icc,omitempty"`
	Init                           bool              `json:"init,omitempty"`
	InitPath                       string            `json:"init-path,omitempty"`
	InsecureRegistries             []string          `json:"insecure-registries,omitempty"`
	IP                             string            `json:"ip,omitempty"`
	IPForward                      bool              `json:"ip-forward,omitempty"`
	IPMasq                         bool              `json:"ip-masq,omitempty"`
	Iptables                       bool              `json:"iptables,omitempty"`
	IP6Tables                      bool              `json:"ip6tables,omitempty"`
	Ipv6                           bool              `json:"ipv6,omitempty"`
	Labels                         []string          `json:"labels,omitempty"`
	LiveRestore                    bool              `json:"live-restore,omitempty"`
	LogDriver                      string            `json:"log-driver,omitempty"`
	LogLevel                       string            `json:"log-level,omitempty"`
	MaxConcurrentDownloads         int               `json:"max-concurrent-downloads,omitempty"`
	MaxConcurrentUploads           int               `json:"max-concurrent-uploads,omitempty"`
	MaxDownloadAttempts            int               `json:"max-download-attempts,omitempty"`
	Mtu                            int               `json:"mtu,omitempty"`
	NoNewPrivileges                bool              `json:"no-new-privileges,omitempty"`
	NodeGenericResources           []string          `json:"node-generic-resources,omitempty"`
	OomScoreAdjust                 int               `json:"oom-score-adjust,omitempty"`
	Pidfile                        string            `json:"pidfile,omitempty"`
	RawLogs                        bool              `json:"raw-logs,omitempty"`
	RegistryMirrors                []string          `json:"registry-mirrors,omitempty"`
	SeccompProfile                 string            `json:"seccomp-profile,omitempty"`
	SelinuxEnabled                 bool              `json:"selinux-enabled,omitempty"`
	ShutdownTimeout                int               `json:"shutdown-timeout,omitempty"`
	StorageDriver                  string            `json:"storage-driver,omitempty"`
	StorageOpts                    []string          `json:"storage-opts,omitempty"`
	SwarmDefaultAdvertiseAddr      string            `json:"swarm-default-advertise-addr,omitempty"`
	TLS                            bool              `json:"tls,omitempty"`
	Tlscacert                      string            `json:"tlscacert,omitempty"`
	Tlscert                        string            `json:"tlscert,omitempty"`
	Tlskey                         string            `json:"tlskey,omitempty"`
	Tlsverify                      bool              `json:"tlsverify,omitempty"`
	UserlandProxy                  bool              `json:"userland-proxy,omitempty"`
	UserlandProxyPath              string            `json:"userland-proxy-path,omitempty"`
	UsernsRemap                    string            `json:"userns-remap,omitempty"`
	ClusterStoreOpts               map[string]string `json:"cluster-store-opts,omitempty"`
	LogOpts                        *DaemonLogOpts    `json:"log-opts,omitempty"`
}

type DaemonLogOpts struct {
	CacheDisabled string `json:"cache-disabled,omitempty"`
	CacheMaxFile  string `json:"cache-max-file,omitempty"`
	CacheMaxSize  string `json:"cache-max-size,omitempty"`
	CacheCompress string `json:"cache-compress,omitempty"`
	Env           string `json:"env,omitempty"`
	Labels        string `json:"labels,omitempty"`
	MaxFile       string `json:"max-file,omitempty"`
	MaxSize       string `json:"max-size,omitempty"`
}
