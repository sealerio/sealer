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

package container_runtime

import (
	"context"
	"fmt"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/utils/os"
	"golang.org/x/sync/errgroup"
	"net"
	"path/filepath"
)

const (
	RemoteAddEtcHosts     = "cat /etc/hosts |grep '%s' || echo '%s' >> /etc/hosts"
	DockerCertDir         = "/etc/docker/certs.d"
	ContainerdCertDir     = "/etc/containerd/certs.d"
	RestartServiceCommand = "systemctl restart %s"
	DockerLoginCommand    = "docker login -u %s -p %s %s && " + KubeletAuthCommand
	NerdctlLoginCommand   = "nerdctl login -u %s -p %s %s && " + KubeletAuthCommand
	KubeletAuthCommand    = "mkdir -p /var/lib/kubelet && cp /root/.docker/config.json /var/lib/kubelet"
	ConfigDaemonCommand   = "sed -i \"s/%s/%s/g\" /etc/docker/daemon.json"
	DefaultEndpoint       = "sea.hub:5000"
)

//Configurator provide configuration Interface for different container runtime.
type Configurator interface {
	// ConfigRegistry config registry via each container runtime ip address
	ConfigRegistry(RegistryConfig, []net.IP) error
}

type RegistryConfig struct {
	SkipTLSVerify      bool
	RegistryDeployHost string
	Endpoint           string
	Domain             string
	Port               string
	Username           string
	Password           string
	CaFile             string
	KeyFile            string
	CertFile           string
}

type DockerRuntimeConfigurator struct {
	rootfs      string
	infraDriver infradriver.InfraDriver
}

func (d DockerRuntimeConfigurator) ConfigRegistry(config RegistryConfig, ips []net.IP) error {
	if err := d.configHostsFile(config.RegistryDeployHost, config.Domain, ips); err != nil {
		return err
	}

	if err := d.configAuthInfo(config.Username, config.Password, config.Endpoint, ips); err != nil {
		return err
	}

	if err := d.configRegistryCert(config.Domain, config.Port, config.CaFile, ips); err != nil {
		return err
	}

	if err := d.configDaemon(config.Endpoint, ips); err != nil {
		return err
	}

	return nil
}

func (d DockerRuntimeConfigurator) configHostsFile(registryIP, domain string, ips []net.IP) error {
	// add registry ip to "/etc/hosts"
	hostsLine := registryIP + " " + domain
	writeToHostsCmd := fmt.Sprintf(RemoteAddEtcHosts, hostsLine, hostsLine)

	f := func(host net.IP) error {
		err := d.infraDriver.CmdAsync(host, writeToHostsCmd)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (d DockerRuntimeConfigurator) configAuthInfo(username, password string, url string, ips []net.IP) error {
	// docker login with username and password
	// cp /root/.docker/config.json to /var/lib/kubelet. make sure kubelet could access registry with credential.
	if username != "" && password != "" {
		return nil
	}

	configAuthCmd := fmt.Sprintf(DockerLoginCommand, username, password, url)

	f := func(host net.IP) error {
		err := d.infraDriver.CmdAsync(host, configAuthCmd)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (d DockerRuntimeConfigurator) configRegistryCert(domain, port string, caFile string, ips []net.IP) error {
	// copy ca cert to "/etc/docker/certs.d/${domain}:${port}/${domain}.crt
	src := filepath.Join(d.rootfs, "certs", caFile)
	dest := filepath.Join(DockerCertDir, domain+":"+port, caFile)

	if !os.IsFileExist(src) {
		return nil
	}

	f := func(host net.IP) error {
		err := d.infraDriver.Copy(host, src, dest)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (d DockerRuntimeConfigurator) configDaemon(endpoint string, ips []net.IP) error {
	// modify daemon.json: "mirrors": ["${endpoint}"],docker server need to restart if modify daemon.json
	if endpoint == DefaultEndpoint {
		return nil
	}

	modifyDaemonFileCmd := fmt.Sprintf(ConfigDaemonCommand, DefaultEndpoint, endpoint)
	restartDockerCommand := fmt.Sprintf(RestartServiceCommand, "docker")
	configDaemonCmd := modifyDaemonFileCmd + " && " + restartDockerCommand

	f := func(host net.IP) error {
		err := d.infraDriver.CmdAsync(host, configDaemonCmd)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (d DockerRuntimeConfigurator) execute(f func(host net.IP) error, ips []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range ips {
		host := ip
		eg.Go(func() error {
			err := f(host)
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func NewDockerRuntimeDriver(rootfs string, infraDriver infradriver.InfraDriver) Configurator {
	return DockerRuntimeConfigurator{
		rootfs:      rootfs,
		infraDriver: infraDriver,
	}
}

type ContainerdRuntimeConfigurator struct {
	rootfs      string
	infraDriver infradriver.InfraDriver
}

func (c ContainerdRuntimeConfigurator) ConfigRegistry(config RegistryConfig, ips []net.IP) error {
	if err := c.configHostsFile(config.RegistryDeployHost, config.Domain, ips); err != nil {
		return err
	}

	if err := c.configRegistryCert(config.Domain, config.Port, config.CaFile, ips); err != nil {
		return err
	}

	if err := c.configAuthInfo(config.Username, config.Password, config.Endpoint, ips); err != nil {
		return err
	}

	return nil
}

func (c ContainerdRuntimeConfigurator) configHostsFile(registryIP, domain string, ips []net.IP) error {
	// add registry ip to "/etc/hosts"
	hostsLine := registryIP + " " + domain
	writeToHostsCmd := fmt.Sprintf(RemoteAddEtcHosts, hostsLine, hostsLine)

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, writeToHostsCmd)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (c ContainerdRuntimeConfigurator) configRegistryCert(domain, port string, caFile string, ips []net.IP) error {
	// copy ca cert to "/etc/containerd/certs.d/${domain}:${port}/${domain}.crt
	src := filepath.Join(c.rootfs, "certs", caFile)
	dest := filepath.Join(ContainerdCertDir, domain+":"+port, caFile)

	if !os.IsFileExist(src) {
		return nil
	}

	f := func(host net.IP) error {
		err := c.infraDriver.Copy(host, src, dest)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func (c ContainerdRuntimeConfigurator) configAuthInfo(username, password string, url string, ips []net.IP) error {
	if username != "" && password != "" {
		return nil
	}

	configAuthCmd := fmt.Sprintf(NerdctlLoginCommand, username, password, url)

	f := func(host net.IP) error {
		err := c.infraDriver.CmdAsync(host, configAuthCmd)
		if err != nil {
			return err
		}
		return nil
	}

	return concurrencyExecute(f, ips)
}

func NewContainerdRuntimeDriver(rootfs string, infraDriver infradriver.InfraDriver) Configurator {
	return ContainerdRuntimeConfigurator{
		rootfs:      rootfs,
		infraDriver: infraDriver,
	}
}

func concurrencyExecute(f func(host net.IP) error, ips []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range ips {
		host := ip
		eg.Go(func() error {
			err := f(host)
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
