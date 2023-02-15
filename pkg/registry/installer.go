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
	"net"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clustercert/cert"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
	netutils "github.com/sealerio/sealer/utils/net"
	osutils "github.com/sealerio/sealer/utils/os"
	strutils "github.com/sealerio/sealer/utils/strings"
)

// Installer provide registry lifecycle management.
type Installer interface {
	// Reconcile registry deploy hosts thought comparing current deploy host and desiredHosts and return the final registry deploy hosts.
	// if current deploy host is less than desiredHosts , means scale-up registry node
	// if current deploy host is bigger than desiredHosts , means scale-down registry node
	// if current deploy host is equal targetHosts , do nothing
	// launch registry node
	// scale-up registry node
	// scale down registry node
	Reconcile(desiredHosts []net.IP) ([]net.IP, error)

	// Clean all registry deploy hosts
	Clean() error
}

func NewInstaller(currentDeployHost []net.IP,
	regConfig *v2.LocalRegistry,
	infraDriver infradriver.InfraDriver,
	distributor imagedistributor.Distributor) Installer {
	return &localInstaller{
		currentDeployHosts: currentDeployHost,
		infraDriver:        infraDriver,
		LocalRegistry:      regConfig,
		distributor:        distributor,
	}
}

type localInstaller struct {
	*v2.LocalRegistry
	currentDeployHosts []net.IP
	infraDriver        infradriver.InfraDriver
	distributor        imagedistributor.Distributor
}

func (l *localInstaller) Reconcile(desiredHosts []net.IP) ([]net.IP, error) {
	// if deployHosts is null,means first time installation
	if len(l.currentDeployHosts) == 0 {
		err := l.install(desiredHosts)
		if err != nil {
			return nil, err
		}
		return desiredHosts, nil
	}

	joinedHosts, deletedHosts := strutils.Diff(l.currentDeployHosts, desiredHosts)
	// if targetHosts is equal deployHosts, just return.
	if len(joinedHosts) == 0 && len(deletedHosts) == 0 {
		return l.currentDeployHosts, nil
	}

	// join new hosts
	if len(joinedHosts) != 0 {
		err := l.install(joinedHosts)
		if err != nil {
			return nil, err
		}
		return append(l.currentDeployHosts, joinedHosts...), nil
	}

	// delete hosts
	if len(deletedHosts) != 0 {
		err := l.clean(deletedHosts)
		if err != nil {
			return nil, err
		}
		return netutils.RemoveIPs(l.currentDeployHosts, deletedHosts), nil
	}

	return nil, nil
}

func (l *localInstaller) install(deployHosts []net.IP) error {
	logrus.Infof("will launch local private registry on %+v\n", deployHosts)

	if err := l.syncBasicAuthFile(deployHosts); err != nil {
		return err
	}

	if err := l.syncRegistryCert(deployHosts); err != nil {
		return err
	}

	if err := l.reconcileRegistry(deployHosts); err != nil {
		return err
	}
	return nil
}

func (l *localInstaller) syncBasicAuthFile(hosts []net.IP) error {
	//gen basic auth info: if not config, will skip.
	if l.RegistryConfig.Username == "" || l.RegistryConfig.Password == "" {
		return nil
	}

	var (
		basicAuthFile = filepath.Join(l.infraDriver.GetClusterRootfsPath(), "etc", common.DefaultRegistryHtPasswdFile)
	)

	if !osutils.IsFileExist(basicAuthFile) {
		htpasswd, err := GenerateHTTPBasicAuth(l.RegistryConfig.Username, l.RegistryConfig.Password)
		if err != nil {
			return err
		}

		err = osutils.NewCommonWriter(basicAuthFile).WriteFile([]byte(htpasswd))
		if err != nil {
			return err
		}
	}

	for _, deployHost := range hosts {
		err := l.infraDriver.Copy(deployHost, basicAuthFile, basicAuthFile)
		if err != nil {
			return fmt.Errorf("failed to copy registry auth file to %s: %v", basicAuthFile, err)
		}
	}

	return nil
}

func (l *localInstaller) syncRegistryCert(hosts []net.IP) error {
	// if deploy registry as InsecureMode ,skip to gen cert.
	if *l.Insecure {
		return nil
	}
	var (
		certPath     = filepath.Join(l.infraDriver.GetClusterRootfsPath(), "certs")
		certName     = l.Domain
		fullCertName = certName + ".crt"
		fullKeyName  = certName + ".key"
	)

	certExisted := osutils.IsFileExist(filepath.Join(certPath, fullCertName))
	keyExisted := osutils.IsFileExist(filepath.Join(certPath, fullKeyName))

	if certExisted && !keyExisted || !certExisted && keyExisted {
		return fmt.Errorf("failed to sync registry cert file %s or %s is not existed", fullCertName, fullKeyName)
	}

	if !certExisted && !keyExisted {
		if err := l.gen(certPath, certName); err != nil {
			return err
		}
	}

	for _, deployHost := range hosts {
		err := l.infraDriver.Copy(deployHost, certPath, certPath)
		if err != nil {
			return fmt.Errorf("failed to copy registry cert to deployHost: %v", err)
		}
	}

	return nil
}

func (l *localInstaller) gen(certPath, certName string) error {
	DNSNames := []string{l.Domain}
	if l.Cert.SubjectAltName != nil {
		DNSNames = append(DNSNames, l.Cert.SubjectAltName.IPs...)
		DNSNames = append(DNSNames, l.Cert.SubjectAltName.DNSNames...)
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

func (l *localInstaller) reconcileRegistry(hosts []net.IP) error {
	var (
		rootfs  = l.infraDriver.GetClusterRootfsPath()
		dataDir = filepath.Join(rootfs, "registry")
	)
	// distribute registry data
	if err := l.distributor.DistributeRegistry(hosts, dataDir); err != nil {
		return err
	}

	// bash init-registry.sh ${port} ${mountData} ${domain}
	clusterEnvs := l.infraDriver.GetClusterEnv()
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", rootfs, strconv.Itoa(l.Port), dataDir, l.Domain)
	for _, deployHost := range hosts {
		if err := l.infraDriver.CmdAsync(deployHost, clusterEnvs, initRegistry); err != nil {
			return err
		}
	}
	return nil
}

func (l *localInstaller) clean(cleanHosts []net.IP) error {
	deleteRegistryCommand := "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
	for _, deployHost := range cleanHosts {
		if err := l.infraDriver.CmdAsync(deployHost, nil, fmt.Sprintf(deleteRegistryCommand, "sealer-registry")); err != nil {
			return err
		}
	}
	return nil
}

func (l *localInstaller) Clean() error {
	return l.clean(l.currentDeployHosts)
}
