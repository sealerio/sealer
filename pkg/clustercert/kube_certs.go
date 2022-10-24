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

package clustercert

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	utilnet "k8s.io/utils/net"

	"github.com/sealerio/sealer/pkg/clustercert/cert"
)

const (

	// KubernetesConfigDir kubernetes default certificate directory
	KubernetesConfigDir = "/etc/kubernetes"

	// KubeDefaultCertPath kubernetes components default certificate directory
	KubeDefaultCertPath = "/etc/kubernetes/pki"

	// KubeDefaultCertEtcdPath etcd default certificate directory
	KubeDefaultCertEtcdPath = "/etc/kubernetes/pki/etcd"
)

type CertificateConfig struct {
	// file saved path only for this cert
	certPath   string
	certName   string
	descriptor *cert.CertificateDescriptor
}

type CertificateConfigFamily struct {
	// default file saved path for this cert Family
	certPath     string
	caConfig     CertificateConfig
	commonConfig []CertificateConfig
}

func (c CertificateConfigFamily) GenerateAll() error {
	var (
		err    error
		caCert *x509.Certificate
		caKey  crypto.Signer
	)

	_, err = os.Stat(cert.PathForCert(c.certPath, c.caConfig.certName))
	if os.IsNotExist(err) {
		caCert, caKey, err = c.generateAuthorityCertificate()
		if err != nil {
			return err
		}
	} else {
		logrus.Info("authority certificate is already exist")
		caCert, caKey, err = c.loadAuthorityCertificate(c.certPath, c.caConfig.certName)
		if err != nil {
			return fmt.Errorf("failed to load an exist cert(%s):  %v", c.caConfig.certName, err)
		}
	}

	err = c.generateCommonCertificate(caCert, caKey)
	if err != nil {
		return err
	}

	return nil
}

func (c CertificateConfigFamily) generateAuthorityCertificate() (*x509.Certificate, crypto.Signer, error) {
	// New authority certificate generator to gen ca cert.
	caGenerator := cert.NewAuthorityCertificateGenerator(*c.caConfig.descriptor)
	caCert, caKey, err := caGenerator.Generate()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate %s cert: %v", c.caConfig.certName, err)
	}

	// write cert file to disk
	err = cert.NewCertificateFileManger(c.certPath, c.caConfig.certName).Write(caCert, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save %s cert: %v", c.caConfig.certName, err)
	}

	return caCert, caKey, nil
}

func (c CertificateConfigFamily) loadAuthorityCertificate(certPath, certName string) (*x509.Certificate, crypto.Signer, error) {
	caCert, caKey, err := cert.NewCertificateFileManger(certPath, certName).Read()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load cert(%s) from disk: %v", certName, err)
	}
	return caCert, caKey, nil
}

func (c CertificateConfigFamily) generateCommonCertificate(caCert *x509.Certificate, caKey crypto.Signer) error {
	for _, config := range c.commonConfig {
		certPath := c.certPath
		if config.certPath != "" {
			certPath = config.certPath
		}

		g, err := cert.NewCommonCertificateGenerator(*config.descriptor, caCert, caKey)
		if err != nil {
			return err
		}

		commonCert, key, err := g.Generate()
		if err != nil {
			return err
		}

		err = cert.NewCertificateFileManger(certPath, config.certName).Write(commonCert, key)
		if err != nil {
			return fmt.Errorf("unable to save %s key: %v", config.certName, err)
		}
	}
	return nil
}

type KubernetesCertService struct {
	kubeCert       CertificateConfigFamily
	etcdCert       CertificateConfigFamily
	frontProxyCert CertificateConfigFamily
	serviceAccount cert.KeyPairFileGenerator
}

func (s KubernetesCertService) GenerateKubeComponentCert() (err error) {
	err = s.kubeCert.GenerateAll()
	if err != nil {
		return err
	}

	err = s.etcdCert.GenerateAll()
	if err != nil {
		return err
	}

	err = s.frontProxyCert.GenerateAll()
	if err != nil {
		return err
	}

	return nil
}

func (s KubernetesCertService) GenerateServiceAccountKeyPair() (err error) {
	err = s.serviceAccount.GenerateAll()
	if err != nil {
		return err
	}

	return nil
}

type Args struct {
	APIServerAltNames cert.AltNames
	NodeName          string
	NodeIP            net.IP
	DNSDomain         string
}

// GenerateAllKubernetesCerts generate all cert.
func GenerateAllKubernetesCerts(certPath, etcdCertPath, nodeName, serviceCIRD, DNSDomain string, altNames []string, nodeIP net.IP) error {
	if certPath == "" || etcdCertPath == "" {
		return fmt.Errorf("must provide cert path")
	}

	if nodeName == "" || nodeIP == nil {
		return fmt.Errorf("must provide node name and node IP")
	}

	if DNSDomain == "" {
		return fmt.Errorf("must provide cluster DNS domain")
	}

	// parse cluster cert args
	clusterCertArgs := Args{
		DNSDomain: DNSDomain,
		NodeIP:    nodeIP,
		NodeName:  nodeName,
		APIServerAltNames: cert.AltNames{
			DNSNames: map[string]string{},
			IPs:      map[string]net.IP{},
		},
	}

	clusterCertArgs.APIServerAltNames.IPs[nodeIP.String()] = nodeIP

	for _, svcCidr := range strings.Split(serviceCIRD, ",") {
		_, svcSubnet, err := net.ParseCIDR(svcCidr)
		if err != nil {
			return errors.Wrapf(err, "unable to parse ServiceSubnet %v", svcCidr)
		}
		svcFirstIP, err := utilnet.GetIndexedIP(svcSubnet, 1)
		if err != nil {
			return err
		}
		clusterCertArgs.APIServerAltNames.IPs[svcFirstIP.String()] = svcFirstIP
	}

	for _, altName := range altNames {
		ip := net.ParseIP(altName)
		if ip != nil {
			clusterCertArgs.APIServerAltNames.IPs[ip.String()] = ip
			continue
		}
		clusterCertArgs.APIServerAltNames.DNSNames[altName] = altName
	}

	// generate all cert.
	certService := KubernetesCertService{
		kubeCert:       getKubeCertificateConfig(certPath, clusterCertArgs.APIServerAltNames, clusterCertArgs.NodeName, clusterCertArgs.DNSDomain),
		etcdCert:       getEtcdCertificateConfig(etcdCertPath, certPath, clusterCertArgs.NodeName, clusterCertArgs.NodeIP),
		frontProxyCert: getFrontProxyCertificateConfig(certPath),
		serviceAccount: cert.NewKeyPairFileGenerator(certPath, "sa"),
	}

	err := certService.GenerateKubeComponentCert()
	if err != nil {
		return err
	}

	err = certService.GenerateServiceAccountKeyPair()
	if err != nil {
		return err
	}

	return nil
}

func getKubeCertificateConfig(certPath string, APIServerAltNames cert.AltNames, nodeName string, DNSDomain string) CertificateConfigFamily {
	kubeCert := CertificateConfigFamily{
		certPath: certPath,
		caConfig: CertificateConfig{
			certName: "ca",
			descriptor: &cert.CertificateDescriptor{
				CommonName:   "kubernetes",
				Organization: nil,
				Year:         100,
				AltNames:     cert.AltNames{},
				Usages:       nil,
			},
		},
		commonConfig: []CertificateConfig{
			{
				certName: "apiserver",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "kube-apiserver",
					Organization: nil,
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				},
			},
			{
				certName: "apiserver-kubelet-client",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "kube-apiserver-kubelet-client",
					Organization: []string{"system:masters"},
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				},
			},
		},
	}

	IPs := map[string]net.IP{
		"127.0.0.1": net.IPv4(127, 0, 0, 1),
	}

	dnsName := map[string]string{
		"localhost":              "localhost",
		"kubernetes":             "kubernetes",
		"kubernetes.default":     "kubernetes.default",
		"kubernetes.default.svc": "kubernetes.default.svc",
		nodeName:                 nodeName,
		fmt.Sprintf("kubernetes.default.svc.%s", DNSDomain): fmt.Sprintf("kubernetes.default.svc.%s", DNSDomain),
	}

	for _, dns := range APIServerAltNames.DNSNames {
		dnsName[dns] = dns
	}

	for _, ip := range APIServerAltNames.IPs {
		IPs[ip.String()] = ip
	}

	for _, config := range kubeCert.commonConfig {
		// add altNames and node name to etcd server cert and peer cert.
		if config.certName == "apiserver" {
			config.descriptor.AltNames.DNSNames = dnsName
			config.descriptor.AltNames.IPs = IPs
			logrus.Info("API server altNames: ", config.descriptor.AltNames)
		}
	}

	return kubeCert
}

func getEtcdCertificateConfig(etcdCertPath, certPath, nodeName string, nodeIP net.IP) CertificateConfigFamily {
	etcdCert := CertificateConfigFamily{
		certPath: etcdCertPath,
		caConfig: CertificateConfig{
			certName: "ca",
			descriptor: &cert.CertificateDescriptor{
				CommonName:   "etcd-ca",
				Organization: nil,
				Year:         100,
				AltNames:     cert.AltNames{},
				Usages:       nil,
			},
		},
		commonConfig: []CertificateConfig{
			{
				certPath: certPath,
				certName: "apiserver-etcd-client",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "kube-apiserver-etcd-client",
					Organization: []string{"system:masters"},
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				},
			},
			{
				certName: "server",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "etcd",
					Organization: nil,
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
				},
			},
			{
				certName: "peer",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "etcd-peer",
					Organization: nil,
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
				},
			},
			{
				certName: "healthcheck-client",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "kube-etcd-healthcheck-client",
					Organization: []string{"system:masters"},
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				},
			},
		},
	}

	altNames := cert.AltNames{
		DNSNames: map[string]string{
			"localhost": "localhost",
			nodeName:    nodeName,
		},
		IPs: map[string]net.IP{
			net.IPv4(127, 0, 0, 1).String(): net.IPv4(127, 0, 0, 1),
			nodeIP.String():                 nodeIP,
			net.IPv6loopback.String():       net.IPv6loopback,
		},
	}

	for _, config := range etcdCert.commonConfig {
		// add altNames and node name to etcd server cert and peer cert.
		if config.certName == "server" || config.certName == "peer" {
			config.descriptor.AltNames = altNames
			config.descriptor.CommonName = nodeName
		}
	}

	return etcdCert
}

func getFrontProxyCertificateConfig(certPath string) CertificateConfigFamily {
	return CertificateConfigFamily{
		certPath: certPath,
		caConfig: CertificateConfig{
			certName: "front-proxy-ca",
			descriptor: &cert.CertificateDescriptor{
				CommonName:   "front-proxy-ca",
				Organization: nil,
				Year:         100,
				AltNames:     cert.AltNames{},
				Usages:       nil,
			},
		},
		commonConfig: []CertificateConfig{
			{
				certName: "front-proxy-client",
				descriptor: &cert.CertificateDescriptor{
					CommonName:   "front-proxy-client",
					Organization: nil,
					Year:         100,
					AltNames:     cert.AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				},
			},
		},
	}
}

// UpdateAPIServerCertSans :renew apiserver cert sans with given ca under pkiPath.
func UpdateAPIServerCertSans(pkiPath string, certSans []string) error {
	baseName := "apiserver"

	APIServerCertSans := cert.AltNames{
		DNSNames: map[string]string{},
		IPs:      map[string]net.IP{},
	}

	for _, altName := range certSans {
		ip := net.ParseIP(altName)
		if ip != nil {
			APIServerCertSans.IPs[ip.String()] = ip
			continue
		}
		APIServerCertSans.DNSNames[altName] = altName
	}

	apiCert, _, err := cert.NewCertificateFileManger(pkiPath, baseName).Read()
	if err != nil {
		return fmt.Errorf("unable to load %s cert: %v", baseName, err)
	}

	for _, dns := range apiCert.DNSNames {
		APIServerCertSans.DNSNames[dns] = dns
	}

	for _, ip := range apiCert.IPAddresses {
		APIServerCertSans.IPs[ip.String()] = ip
	}

	apiCertConfig := cert.CertificateDescriptor{
		Year:         100,
		CommonName:   apiCert.Subject.CommonName,
		Organization: apiCert.Subject.Organization,
		AltNames:     APIServerCertSans,
		Usages:       apiCert.ExtKeyUsage,
	}

	// load ca cert form pkiPath
	caCert, caKey, err := cert.NewCertificateFileManger(pkiPath, "ca").Read()
	if err != nil {
		return fmt.Errorf("unable to load %s cert: %v", baseName, err)
	}

	if err != nil {
		return err
	}

	// new api server cert
	generator, err := cert.NewCommonCertificateGenerator(apiCertConfig, caCert, caKey)
	if err != nil {
		return fmt.Errorf("unable to generate %s cert: %v", baseName, err)
	}

	newCert, newKey, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("unable to generate %s cert: %v", baseName, err)
	}

	// write cert file to disk
	err = cert.NewCertificateFileManger(pkiPath, baseName).Write(newCert, newKey)
	if err != nil {
		return fmt.Errorf("unable to save %s cert: %v", baseName, err)
	}

	return nil
}
