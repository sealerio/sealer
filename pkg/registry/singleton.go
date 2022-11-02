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
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containers/common/pkg/auth"
	"github.com/pelletier/go-toml"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clustercert/cert"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	osutils "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/os/fs"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/sirupsen/logrus"
)

const (
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
)

type localSingletonConfigurator struct {
	*LocalRegistry

	containerRuntimeInfo containerruntime.Info
	infraDriver          infradriver.InfraDriver
	distributor          imagedistributor.Distributor
}

// Clean will stop local private registry via rootfs scripts.
func (c *localSingletonConfigurator) Clean() error {
	//TODO delete local registry by container runtime sdk.
	deleteRegistryCommand := "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"

	return c.infraDriver.CmdAsync(c.DeployHost, fmt.Sprintf(deleteRegistryCommand, "sealer-registry"))
}

func (c *localSingletonConfigurator) UninstallFrom(hosts []net.IP) error {
	var uninstallCmd []string
	if c.Auth.Username != "" && c.Auth.Password != "" {
		//todo use sdk to logout instead of shell cmd
		logoutCmd := fmt.Sprintf("docker logout %s ", c.Domain+":"+strconv.Itoa(c.Port))
		if c.containerRuntimeInfo.Type != "docker" {
			logoutCmd = fmt.Sprintf("nerdctl logout %s ", c.Domain+":"+strconv.Itoa(c.Port))
		}
		uninstallCmd = append(uninstallCmd, logoutCmd)
	}

	uninstallCmd = append(uninstallCmd, shellcommand.CommandUnSetHostAlias(shellcommand.DefaultSealerHostAlias))
	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, strings.Join(uninstallCmd, "&&"))
		if err != nil {
			return fmt.Errorf("failed to delete registry configuration: %v", err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localSingletonConfigurator) GetDriver() (Driver, error) {
	// todo transfer registry config to driver more precisely obey to minimization principle
	endpoint := c.Domain + ":" + strconv.Itoa(c.Port)
	return newLocalRegistryDriver(endpoint, c.DataDir, c.DeployHost, c.infraDriver, c.distributor), nil
}

// Launch will start local private registry via rootfs scripts.
func (c *localSingletonConfigurator) Launch() error {
	if err := c.genBasicAuth(); err != nil {
		return err
	}

	if err := c.genRegistryCert(); err != nil {
		return err
	}

	if err := c.reconcileRegistry(); err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) genBasicAuth() error {
	//gen basic auth info: if not config, will skip.
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return nil
	}

	var (
		localBasicAuthFile = filepath.Join("/tmp", DefaultRegistryHtPasswdFile)
		basicAuthFile      = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "etc", DefaultRegistryHtPasswdFile)
	)

	htpasswd, err := GenerateHTTPBasicAuth(c.Auth.Username, c.Auth.Password)
	if err != nil {
		return err
	}

	err = osutils.NewCommonWriter(localBasicAuthFile).WriteFile([]byte(htpasswd))
	if err != nil {
		return err
	}

	err = c.infraDriver.Copy(c.DeployHost, localBasicAuthFile, basicAuthFile)
	if err != nil {
		return fmt.Errorf("failed to copy registry auth file to %s: %v", basicAuthFile, err)
	}

	return fs.FS.RemoveAll(localBasicAuthFile)
}

func (c *localSingletonConfigurator) genRegistryCert() error {
	// if deploy registry as InsecureMode ,skip to gen cert.
	if c.InsecureMode {
		return nil
	}

	var (
		localCertPath = filepath.Join("/tmp", "certs")
		certPath      = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "certs")
		certName      = c.Domain
		fullCertName  = certName + ".crt"
		fullKeyName   = certName + ".key"
	)

	certExisted, err := c.infraDriver.IsFileExist(c.DeployHost, filepath.Join(certPath, fullCertName))
	if err != nil {
		return err
	}

	keyExisted, err := c.infraDriver.IsFileExist(c.DeployHost, filepath.Join(certPath, fullKeyName))
	if err != nil {
		return err
	}

	if !certExisted || !keyExisted {
		if err = c.gen(localCertPath, certName); err != nil {
			return err
		}

		err = c.infraDriver.Copy(c.DeployHost, localCertPath, certPath)
		if err != nil {
			return fmt.Errorf("failed to copy registry cert to deployHost: %v", err)
		}
		err = fs.FS.RemoveAll(localCertPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *localSingletonConfigurator) gen(certPath, certName string) error {
	DNSNames := []string{c.Domain}
	if c.Cert.SubjectAltName != nil {
		DNSNames = append(DNSNames, c.Cert.SubjectAltName.IPs...)
		DNSNames = append(DNSNames, c.Cert.SubjectAltName.DNSNames...)
	}

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

func (c *localSingletonConfigurator) reconcileRegistry() error {
	var (
		rootfs     = c.infraDriver.GetClusterRootfsPath()
		dataDir    = c.DataDir
		deployHost = c.DeployHost
	)

	if err := c.distributor.DistributeRegistry(deployHost, dataDir); err != nil {
		return err
	}

	// bash init-registry.sh ${port} ${mountData} ${domain}
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", rootfs, strconv.Itoa(c.Port), dataDir, c.Domain)
	if err := c.infraDriver.CmdAsync(c.DeployHost, initRegistry); err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) InstallOn(hosts []net.IP) error {
	if err := c.configureHostsFile(hosts); err != nil {
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

func (c *localSingletonConfigurator) configureHostsFile(hosts []net.IP) error {
	// add registry ip to "/etc/hosts"
	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, shellcommand.CommandSetHostAlias(c.Domain, c.DeployHost.String(), shellcommand.DefaultSealerHostAlias))
		if err != nil {
			return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localSingletonConfigurator) configureRegistryCert(hosts []net.IP) error {
	// if deploy registry as InsecureMode ,skip to configure cert.
	if c.InsecureMode {
		return nil
	}

	var (
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
		caFile   = c.Domain + ".crt"
		src      = filepath.Join(c.infraDriver.GetClusterRootfsPath(), "certs", caFile)
		dest     = filepath.Join(c.containerRuntimeInfo.CertsDir, endpoint, caFile)
	)

	return c.copy2RemoteHosts(src, dest, hosts)
}

func (c *localSingletonConfigurator) configureAccessCredential(hosts []net.IP) error {
	var (
		username        = c.Auth.Username
		password        = c.Auth.Password
		endpoint        = c.Domain + ":" + strconv.Itoa(c.Port)
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

func (c *localSingletonConfigurator) copy2RemoteHosts(src, dest string, hosts []net.IP) error {
	f := func(host net.IP) error {
		err := c.infraDriver.Copy(host, src, dest)
		if err != nil {
			return fmt.Errorf("failed to copy local file %s to remote %s : %v", src, dest, err)
		}
		return nil
	}

	return c.infraDriver.Execute(hosts, f)
}

func (c *localSingletonConfigurator) configureDaemonService(hosts []net.IP) error {
	var (
		src      string
		dest     string
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
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

func (c *localSingletonConfigurator) configureDockerDaemonService(endpoint, daemonFile string) error {
	var daemonConf DaemonConfig

	b, err := ioutil.ReadFile(filepath.Clean(daemonFile))
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

	return osutils.NewCommonWriter(daemonFile).WriteFile(content)
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
	return osutils.NewCommonWriter(hostTomlFile).WriteFile([]byte(tree.String()))
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
