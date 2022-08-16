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
	"path/filepath"
)

const (
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DeleteRegistryCommand       = "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
)

type localSingletonConfigurator struct {
	rootfs string
	LocalRegistry
	infraDriver          infradriver.InfraDriver
	ContainerRuntimeInfo containerruntime.Info
}

// Reconcile local private registry by rootfs scripts.
func (c *localSingletonConfigurator) Reconcile() (Driver, error) {

	if err := c.genTLSCerts(); err != nil {
		return nil, err
	}

	if err := c.genBasicAuth(); err != nil {
		return nil, err
	}

	if err := c.reconcile(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *localSingletonConfigurator) genTLSCerts() error {
	//gen tls cert by default
	registryCertPath := filepath.Join(c.rootfs, "certs")
	baseName := "ca"

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
	err = cert.NewCertificateFileManger(registryCertPath, baseName).Write(caCert, caKey)
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

	var basicAuthFile = filepath.Join(c.rootfs, "etc", DefaultRegistryHtPasswdFile)

	htpasswd, err := GenerateHTTPBasicAuth(c.Auth.Username, c.Auth.Password)
	if err != nil {
		return err
	}

	writeCMD := fmt.Sprintf("echo '%s' > %s", htpasswd, basicAuthFile)

	err = c.infraDriver.CmdAsync(c.DeployHost, writeCMD)
	if err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) reconcile() error {
	// bash init-registry.sh ${port} ${mountData} ${domain}
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", c.rootfs, c.Port, c.DataDir, c.Domain)
	if err := c.infraDriver.CmdAsync(c.DeployHost, initRegistry); err != nil {
		return err
	}

	return nil
}
