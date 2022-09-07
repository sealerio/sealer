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
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/sealerio/sealer/pkg/clustercert/cert"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/shellcommand"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
)

const (
	DefaultDomain               = "sea.hub"
	DefaultPort                 = 5000
	DefaultEndpoint             = "sea.hub:5000"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
)

type localSingletonConfigurator struct {
	*LocalRegistry

	containerRuntimeInfo containerruntime.Info
	infraDriver          infradriver.InfraDriver
}

// Clean will stop local private registry via rootfs scripts.
func (c *localSingletonConfigurator) Clean() error {
	//TODO delete local registry by container runtime sdk.
	deleteRegistryCommand := "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"

	return c.infraDriver.CmdAsync(c.DeployHost, fmt.Sprintf(deleteRegistryCommand, "sealer-registry"))
}

func (c *localSingletonConfigurator) GetDriver() (Driver, error) {
	return newLocalRegistryDriver(c.DataDir, c.infraDriver), nil
}

// Reconcile will start local private registry via rootfs scripts.
func (c *localSingletonConfigurator) Reconcile() error {
	if err := c.genTLSCerts(); err != nil {
		return err
	}

	if err := c.genBasicAuth(); err != nil {
		return err
	}

	if err := c.initRegistry(); err != nil {
		return err
	}

	if err := c.configureHostsFile(); err != nil {
		return err
	}

	if err := c.configureRegistryCert(); err != nil {
		return err
	}

	if err := c.configureDaemonService(); err != nil {
		return err
	}

	if err := c.configureKubeletAuthInfo(); err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) genTLSCerts() error {
	// if deploy registry as InsecureMode ,skip to gen cert.
	if c.InsecureMode {
		return nil
	}

	var (
		certPath = filepath.Join(c.infraDriver.GetClusterRootfs(), "certs")
		certName = c.Domain
	)

	DNSNames := []string{c.Domain}
	DNSNames = append(DNSNames, c.Cert.SubjectAltName.IPs...)
	DNSNames = append(DNSNames, c.Cert.SubjectAltName.DNSNames...)

	regCertConfig := cert.CertificateDescriptor{
		CommonName:   "registry-ca",
		DNSNames:     DNSNames,
		Organization: nil,
		Year:         100,
		AltNames:     cert.AltNames{},
		Usages:       nil,
	}

	caGenerator := cert.NewAuthorityCertificateGenerator(regCertConfig)
	caCert, caKey, err := caGenerator.Generate()
	if err != nil {
		return fmt.Errorf("unable to generate registry cert: %v", err)
	}

	// write cert file to disk
	err = cert.NewCertificateFileManger(certPath, certName).Write(caCert, caKey)
	if err != nil {
		return fmt.Errorf("unable to save registry cert: %v", err)
	}

	return nil
}

func (c *localSingletonConfigurator) genBasicAuth() error {
	//gen basic auth info: if not config, will skip.
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return nil
	}

	var (
		basicAuthFile = filepath.Join(c.infraDriver.GetClusterRootfs(), "etc", DefaultRegistryHtPasswdFile)
	)

	htpasswd, err := GenerateHTTPBasicAuth(c.Auth.Username, c.Auth.Password)
	if err != nil {
		return err
	}

	return os.NewCommonWriter(basicAuthFile).WriteFile([]byte(htpasswd))
}

func (c *localSingletonConfigurator) initRegistry() error {
	// bash init-registry.sh ${port} ${mountData} ${domain}
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", c.infraDriver.GetClusterRootfs(), c.Port, c.DataDir, c.Domain)
	if err := c.infraDriver.CmdAsync(c.DeployHost, initRegistry); err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) configureHostsFile() error {
	// add registry ip to "/etc/hosts"
	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, shellcommand.CommandSetHostAlias(c.Domain, c.DeployHost.String()))
		if err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}
		return nil
	}

	return c.infraDriver.ConcurrencyExecute(f)
}

func (c *localSingletonConfigurator) configureKubeletAuthInfo() error {
	var (
		username = c.Auth.Username
		password = c.Auth.Username
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
	)

	if username == "" || password == "" {
		return nil
	}

	configAuthCmd := fmt.Sprintf("nerdctl login -u %s -p %s %s && mkdir -p /var/lib/kubelet && cp /root/.docker/config.json /var/lib/kubelet",
		username, password, endpoint)

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, configAuthCmd)
		if err != nil {
			return fmt.Errorf("failed to config kubelet auth, cmd is %s: %v", configAuthCmd, err)
		}
		return nil
	}

	return c.infraDriver.ConcurrencyExecute(f)
}

func (c *localSingletonConfigurator) configureRegistryCert() error {
	// copy ca cert to "/etc/containerd/certs.d/${domain}:${port}/${domain}.crt
	var (
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
		caFile   = c.Domain + ".crt"
		dest     = filepath.Join(c.containerRuntimeInfo.CertsDir, endpoint, caFile)
		src      = filepath.Join(c.infraDriver.GetClusterRootfs(), "certs", caFile)
	)

	if !os.IsFileExist(src) {
		return nil
	}

	f := func(host net.IP) error {
		err := c.infraDriver.Copy(host, src, dest)
		if err != nil {
			return fmt.Errorf("failed to copy registry cert %s: %v", src, err)
		}
		return nil
	}

	return c.infraDriver.ConcurrencyExecute(f)
}

func (c *localSingletonConfigurator) configureDaemonService() error {
	var (
		src      string
		dest     string
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
	)

	if endpoint == DefaultEndpoint {
		return nil
	}

	if c.containerRuntimeInfo.Config.Type == "docker" {
		src = filepath.Join(c.infraDriver.GetClusterRootfs(), "etc", "daemon.json")
		dest = "/etc/docker/daemon.json"
		if err := c.configureDockerDaemonService(endpoint, src); err != nil {
			return err
		}
	}

	if c.containerRuntimeInfo.Config.Type == "containerd" {
		src = filepath.Join(c.infraDriver.GetClusterRootfs(), "etc", "hosts.toml")
		dest = filepath.Join("/etc/containerd/certs.d", endpoint, "hosts.toml")
		if err := c.configureContainerdDaemonService(endpoint, src); err != nil {
			return err
		}
	}

	// for docker: copy daemon.json to "/etc/docker/daemon.json"
	// for containerd : copy hosts.toml to "/etc/containerd/certs.d/${domain}:${port}/hosts.toml"
	for _, ip := range c.infraDriver.GetHostIPList() {
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

func (c *localSingletonConfigurator) configureDockerDaemonService(endpoint, daemonFile string) error {
	var daemonConf DaemonConfig

	b, err := ioutil.ReadFile(daemonFile)
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

	daemonConf.MirrorRegistries = append(daemonConf.MirrorRegistries, MirrorRegistry{
		Domain:  "*",
		Mirrors: []string{"https://" + endpoint},
	})

	content, err := json.Marshal(daemonConf)

	if err != nil {
		return fmt.Errorf("failed to marshal daemonFile: %v", err)
	}

	return os.NewCommonWriter(daemonFile).WriteFile(content)
}

func (c *localSingletonConfigurator) configureContainerdDaemonService(endpoint, hostTomlFile string) error {
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
	return os.NewCommonWriter(hostTomlFile).WriteFile([]byte(tree.String()))
}

type DaemonConfig struct {
	MirrorRegistries               []MirrorRegistry  `json:"mirror-registries,omitempty"`
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

type MirrorRegistry struct {
	Domain  string   `json:"domain,omitempty"`
	Mirrors []string `json:"mirrors,omitempty"`
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
