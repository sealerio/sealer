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
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/sealerio/sealer/pkg/clustercert/cert"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/shellcommand"
	"net"
	"path/filepath"
	"strconv"
)

const (
	DefaultEndpoint = "sea.hub:5000"

	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DeleteRegistryCommand       = "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
)

type localSingletonConfigurator struct {
	*LocalRegistry

	containerRuntimeInfo containerruntime.Info
	infraDriver          infradriver.InfraDriver
}

// Reconcile local private registry by rootfs scripts.
func (c *localSingletonConfigurator) Reconcile() (Driver, error) {
	if err := c.genTLSCerts(); err != nil {
		return nil, err
	}

	if err := c.genBasicAuth(); err != nil {
		return nil, err
	}

	if err := c.initRegistry(); err != nil {
		return nil, err
	}

	if err := c.configHostsFile(); err != nil {
		return nil, err
	}

	if err := c.configKubeletAuthInfo(); err != nil {
		return nil, err
	}

	if err := c.configRegistryCert(); err != nil {
		return nil, err
	}

	if err := c.configDaemonService(); err != nil {
		return nil, err
	}

	return NewLocalRegistryDriver(c.DataDir, c.infraDriver), nil
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

func (c *localSingletonConfigurator) configHostsFile() error {
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

func (c *localSingletonConfigurator) configKubeletAuthInfo() error {
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

func (c *localSingletonConfigurator) configRegistryCert() error {
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

func (c *localSingletonConfigurator) configDaemonService() error {
	var (
		src      string
		dest     string
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
	)

	if endpoint == DefaultEndpoint {
		return nil
	}

	if c.containerRuntimeInfo.Type == "docker" {
		src = filepath.Join(c.infraDriver.GetClusterRootfs(), "etc", "daemon.json")
		dest = "/etc/docker/daemon.json"
		if err := c.configDockerDaemonService(endpoint, src); err != nil {
			return err
		}
	}

	if c.containerRuntimeInfo.Type == "containerd" {
		src = filepath.Join(c.infraDriver.GetClusterRootfs(), "etc", "hosts.toml")
		dest = filepath.Join("/etc/containerd/certs.d", endpoint, "hosts.toml")
		if err := c.configContainerdDaemonService(endpoint, src); err != nil {
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

func (c *localSingletonConfigurator) configDockerDaemonService(endpoint, daemonFile string) error {
	daemonConf := MirrorRegistryConfig{
		MirrorRegistries: []MirrorRegistry{
			MirrorRegistry{
				Domain:  "*",
				Mirrors: []string{"https://" + endpoint},
			},
		},
	}

	content, err := json.Marshal(daemonConf)

	if err != nil {
		return fmt.Errorf("failed to marshal daemonFile: %v", err)
	}

	return os.NewCommonWriter(daemonFile).WriteFile(content)
}

func (c *localSingletonConfigurator) configContainerdDaemonService(endpoint, hostTomlFile string) error {
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

type MirrorRegistryConfig struct {
	MirrorRegistries []MirrorRegistry `json:"mirror-registries"`
}

type MirrorRegistry struct {
	Domain  string   `json:"domain"`
	Mirrors []string `json:"mirrors"`
}
