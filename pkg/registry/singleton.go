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
	"fmt"
	"github.com/sealerio/sealer/pkg/clustercert/cert"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/os"
	"net"
	"path/filepath"
	"strconv"
)

const (
	DockerCertDir               = "/etc/docker/certs.d"
	ContainerdCertDir           = "/etc/containerd/certs.d"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DeleteRegistryCommand       = "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
)

type localSingletonConfigurator struct {
	rootfs string
	*LocalRegistry

	configFileGenerator          ConfigFileGenerator
	containerRuntimeConfigurator containerruntime.Configurator
	infraDriver                  infradriver.InfraDriver
	containerRuntimeInfo         containerruntime.Info
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

	config := containerruntime.DaemonConfig{
		Endpoint: c.Domain + ":" + strconv.Itoa(c.Port),
	}

	if err := c.containerRuntimeConfigurator.ConfigDaemonService(config); err != nil {
		return nil, err
	}

	return NewLocalRegistryDriver(c.DataDir, c.infraDriver), nil
}

func (c *localSingletonConfigurator) genTLSCerts() error {
	// if deploy registry as InsecureMode ,skip to gen cert.
	if c.InsecureMode {
		return nil
	}

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

	return c.configFileGenerator.GenTLSCert(c.Domain, regCertConfig)
}

func (c *localSingletonConfigurator) genBasicAuth() error {
	//gen basic auth info: if not config, will skip.
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return nil
	}

	return c.configFileGenerator.GenBasicAuth(c.Auth.Username, c.Auth.Password)
}

func (c *localSingletonConfigurator) initRegistry() error {
	// bash init-registry.sh ${port} ${mountData} ${domain}
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", c.rootfs, c.Port, c.DataDir, c.Domain)
	if err := c.infraDriver.CmdAsync(c.DeployHost, initRegistry); err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) configHostsFile() error {
	// add registry ip to "/etc/hosts"
	var (
		registryIP = c.DeployHost.String()
		domain     = c.Domain
		ips        = c.infraDriver.GetHostIPList()
	)

	hostsLine := registryIP + " " + domain
	writeToHostsCmd := fmt.Sprintf("cat /etc/hosts |grep '%s' || echo '%s' >> /etc/hosts", hostsLine, hostsLine)

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, writeToHostsCmd)
		if err != nil {
			return fmt.Errorf("failed to exec cmd %s: %v", writeToHostsCmd, err)
		}
		return nil
	}

	return ConcurrencyExecute(f, ips)
}

func (c *localSingletonConfigurator) configKubeletAuthInfo() error {
	var (
		username = c.Auth.Username
		password = c.Auth.Username
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
		ips      = c.infraDriver.GetHostIPList()
	)

	if username == "" || password == "" {
		return nil
	}

	configAuthCmd := fmt.Sprintf("nerdctl login -u %s -p %s %s && mkdir -p /var/lib/kubelet && cp /root/.docker/config.json /var/lib/kubelet",
		username, password, endpoint)

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, configAuthCmd)
		if err != nil {
			return fmt.Errorf("failed to exec cmd %s: %v", configAuthCmd, err)
		}
		return nil
	}

	return ConcurrencyExecute(f, ips)
}

func (c *localSingletonConfigurator) configRegistryCert() error {
	var (
		endpoint = c.Domain + ":" + strconv.Itoa(c.Port)
		caFile   = c.Domain + ".crt"
		ips      = c.infraDriver.GetHostIPList()
		dest     = filepath.Join(DockerCertDir, endpoint, caFile)
		src      = filepath.Join(c.rootfs, "certs", caFile)
	)

	// copy ca cert to "/etc/containerd/certs.d/${domain}:${port}/${domain}.crt
	if c.containerRuntimeInfo.Type == "containerd" {
		dest = filepath.Join(ContainerdCertDir, endpoint, caFile)
	}

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

	return ConcurrencyExecute(f, ips)
}

// ConfigFileGenerator gen registry setting files,like basic auth file Or cert files.
type ConfigFileGenerator interface {
	GenBasicAuth(username, password string) error
	GenTLSCert(certName string, certData cert.CertificateDescriptor) error
}

type localFileGenerator struct {
	basicAuthFile string
	certPath      string
}

func (l localFileGenerator) GenBasicAuth(username, password string) error {
	htpasswd, err := GenerateHTTPBasicAuth(username, password)
	if err != nil {
		return err
	}

	return os.NewCommonWriter(l.basicAuthFile).WriteFile([]byte(htpasswd))
}

func (l localFileGenerator) GenTLSCert(certName string, certData cert.CertificateDescriptor) error {
	caGenerator := cert.NewAuthorityCertificateGenerator(certData)
	caCert, caKey, err := caGenerator.Generate()
	if err != nil {
		return fmt.Errorf("unable to generate registry cert: %v", err)
	}

	// write cert file to disk
	err = cert.NewCertificateFileManger(l.certPath, certName).Write(caCert, caKey)
	if err != nil {
		return fmt.Errorf("unable to save registry cert: %v", err)
	}

	return nil
}

func NewLocalFileGenerator(rootfs string) ConfigFileGenerator {
	return localFileGenerator{
		basicAuthFile: filepath.Join(rootfs, "etc", DefaultRegistryHtPasswdFile),
		certPath:      filepath.Join(rootfs, "certs"),
	}
}

type RemoteFileSystemConfigurator struct {
}
