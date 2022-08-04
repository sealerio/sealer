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

package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	utilnet "k8s.io/utils/net"
	"math"
	"math/big"
	"net"
	"os"
	"path"
	"time"
)

// KeyPairFileGenerator write symmetric encryption key, like: sa.key and sa.pub
type KeyPairFileGenerator struct {
	path string
	name string
}

func NewKeyPairFileGenerator(certPath string, certName string) KeyPairFileGenerator {
	return KeyPairFileGenerator{
		path: certPath,
		name: certName,
	}
}

func (k KeyPairFileGenerator) GenerateAll() error {
	_, err := os.Stat(path.Join(k.path, "sa.key"))
	if !os.IsNotExist(err) {
		logrus.Info("sa.key sa.pub already exist")
		return nil
	}

	key, err := NewPrivateKey(x509.RSA)
	if err != nil {
		return err
	}

	err = k.writePrivateKey(key)
	if err != nil {
		return err
	}

	return k.writePublicKey(key.Public())
}

func (k KeyPairFileGenerator) writePrivateKey(key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(k.path, k.name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM %v", err)
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %s %v", privateKeyPath, err)
	}

	return nil
}

func (k KeyPairFileGenerator) writePublicKey(key crypto.PublicKey) error {
	if key == nil {
		return errors.New("public key cannot be nil when writing to file")
	}

	publicKeyBytes, err := EncodePublicKeyPEM(key)
	if err != nil {
		return err
	}
	publicKeyPath := pathForPublicKey(k.path, k.name)
	if err := keyutil.WriteKey(publicKeyPath, publicKeyBytes); err != nil {
		return fmt.Errorf("unable to write public key to file %s %v", publicKeyPath, err)
	}

	return nil
}

//CertificateFileManger Asymmetric encryption, like ca.crt and ca.key
type CertificateFileManger interface {
	WriteKey(key crypto.Signer) error
	WriteCert(cert *x509.Certificate) error
	ReadKey() (key crypto.Signer, err error)
	ReadCert() (cert *x509.Certificate, err error)
}

type CertificateFile struct {
	certName string
	certPath string
}

func (c CertificateFile) WriteKey(key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(c.certPath, c.certName)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM %v", err)
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %s %v", privateKeyPath, err)
	}

	return nil
}

func (c CertificateFile) WriteCert(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	certificatePath := pathForCert(c.certPath, c.certName)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return fmt.Errorf("unable to write certificate to file %s %v", certificatePath, err)
	}

	return nil
}

func (c CertificateFile) ReadKey() (crypto.Signer, error) {
	// Parse the private key from a file
	privateKey, err := keyutil.PrivateKeyFromFile(pathForKey(c.certPath, c.certName))
	if err != nil {
		return nil, fmt.Errorf("couldn't load the private key file (%s): %v", privateKey, err)
	}

	// Allow RSA and ECDSA formats only
	var key crypto.Signer
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		key = k
	case *ecdsa.PrivateKey:
		key = k
	default:
		return nil, fmt.Errorf("couldn't convert the private key file %v", err)
	}

	return key, nil
}

func (c CertificateFile) ReadCert() (cert *x509.Certificate, err error) {
	certs, err := certutil.CertsFromFile(pathForCert(c.certPath, c.certName))
	if err != nil {
		return nil, err
	}
	return certs[0], nil
}

func NewCertificateFileManger(certPath string, certName string) CertificateFileManger {
	return CertificateFile{
		certName: certName,
		certPath: certPath,
	}
}

// CertificateDescriptor contains the basic fields required for creating a certificate
type CertificateDescriptor struct {
	CommonName   string
	DNSNames     []string
	Organization []string
	Year         time.Duration
	AltNames     AltNames
	Usages       []x509.ExtKeyUsage
}

type CertificateGenerator interface {
	Generate() (*x509.Certificate, crypto.Signer, error)
}

type AuthorityCertificateGenerator struct {
	certName   string
	config     CertificateDescriptor
	fileManger CertificateFileManger
}

func (m AuthorityCertificateGenerator) Generate() (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(x509.UnknownPublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating CA certificate (%s): %v",
			m.certName, err)
	}

	cert, err := m.generateSelfSignedCACert(key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate %s cert %s", m.certName, err)
	}

	// write cert file to disk
	err = m.fileManger.WriteCert(cert)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save %s cert %s", m.certName, err)
	}

	// write key file to disk
	err = m.fileManger.WriteKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save %s key %s", m.certName, err)
	}

	return cert, key, nil
}

func (m AuthorityCertificateGenerator) generateSelfSignedCACert(key crypto.Signer) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   m.config.CommonName,
			Organization: m.config.Organization,
		},
		DNSNames:              m.config.DNSNames,
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d * m.config.Year).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewAuthorityCertificateGenerator(certPath string, certName string, config CertificateDescriptor) CertificateGenerator {
	return AuthorityCertificateGenerator{
		certName:   certName,
		fileManger: NewCertificateFileManger(certPath, certName),
		config:     config,
	}
}

type CommonCertificateGenerator struct {
	certName   string
	fileManger CertificateFileManger
	config     CertificateDescriptor
	caCert     *x509.Certificate
	caKey      crypto.Signer
}

func (m CommonCertificateGenerator) Generate() (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(x509.UnknownPublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating common certificate (%s): %v",
			m.certName, err)
	}

	cert, err := m.generateSignedCert(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate signed cert (%s): %v", m.certName, err)
	}

	// write cert file to disk
	err = m.fileManger.WriteCert(cert)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save %s cert: %v", m.certName, err)
	}

	// write key file to disk
	err = m.fileManger.WriteKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save %s key: %v", m.certName, err)
	}

	return cert, key, nil
}

// generateSignedCert creates a signed certificate using the given CA certificate and key
func (m CommonCertificateGenerator) generateSignedCert(key crypto.Signer) (*x509.Certificate, error) {
	var dnsNames []string
	var ips []net.IP

	for _, v := range m.config.AltNames.DNSNames {
		dnsNames = append(dnsNames, v)
	}
	for _, v := range m.config.AltNames.IPs {
		ips = append(ips, v)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   m.config.CommonName,
			Organization: m.config.Organization,
		},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
		SerialNumber: serial,
		NotBefore:    m.caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d * m.config.Year).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  m.config.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, m.caCert, key.Public(), m.caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewCommonCertificateGenerator(certPath string, certName string, config CertificateDescriptor, caCert *x509.Certificate, caKey crypto.Signer) (CertificateGenerator, error) {
	if config.CommonName == "" {
		return nil, errors.New("must specify a CommonName for cert")
	}

	if len(config.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	return CommonCertificateGenerator{
		certName:   certName,
		fileManger: NewCertificateFileManger(certPath, certName),
		config:     config,
		caCert:     caCert,
		caKey:      caKey,
	}, nil
}

type CertificateConfig struct {
	certName   string
	descriptor *CertificateDescriptor
}

type CertificateConfigFamily struct {
	certPath     string
	caConfig     CertificateConfig
	commonConfig []CertificateConfig
}

func (c CertificateConfigFamily) GenerateAll() error {
	caGenerator := NewAuthorityCertificateGenerator(c.certPath, c.caConfig.certName, *c.caConfig.descriptor)
	caCert, caKey, err := caGenerator.Generate()

	if err != nil {
		return err
	}

	for _, config := range c.commonConfig {
		g, err := NewCommonCertificateGenerator(c.certPath, config.certName, *config.descriptor, caCert, caKey)
		if err != nil {
			return err
		}
		_, _, err = g.Generate()
		if err != nil {
			return err
		}
	}

	return nil
}

func GetKubeCertificateConfig(certPath string, APIServerAltNames AltNames, nodeName string, DNSDomain string) CertificateConfigFamily {
	for _, dns := range APIServerAltNames.DNSNames {
		(*certList)[APIserverCert].AltNames.DNSNames[dns] = dns
	}

	svcDNS := fmt.Sprintf("kubernetes.default.svc.%s", DNSDomain)
	(*certList)[APIserverCert].AltNames.DNSNames[svcDNS] = svcDNS
	(*certList)[APIserverCert].AltNames.DNSNames[nodeName] = nodeName

	for _, ip := range APIServerAltNames.IPs {
		(*certList)[APIserverCert].AltNames.IPs[ip.String()] = ip
	}
	logrus.Info("API server altNames : ", (*certList)[APIserverCert].AltNames)
}

func GetEtcdCertificateConfig(certPath string, nodeName string, nodeIP net.IP) CertificateConfigFamily {
	altname := AltNames{
		DNSNames: map[string]string{
			"localhost":   "localhost",
			nodeName:nodeName,
		},
		IPs: map[string]net.IP{
			net.IPv4(127, 0, 0, 1).String(): net.IPv4(127, 0, 0, 1),
			nodeIP.To4().String():      nodeIP.To4(),
			net.IPv6loopback.String():       net.IPv6loopback,
		},
	}

	(*certList)[EtcdServerCert].CommonName = nodeName
	(*certList)[EtcdServerCert].AltNames = altname
	(*certList)[EtcdPeerCert].CommonName = nodeName
	(*certList)[EtcdPeerCert].AltNames = altname

	logrus.Infof("Etcd altnames : %v, commonName : %s", (*certList)[EtcdPeerCert].AltNames, (*certList)[EtcdPeerCert].CommonName)


}

func GetFrontProxyCertificateConfig(certPath string) CertificateConfigFamily {
	return CertificateConfigFamily{
		certPath: certPath,
		caConfig: CertificateConfig{
			certName: "front-proxy-ca",
			descriptor: &CertificateDescriptor{
				CommonName:   "front-proxy-ca",
				Organization: nil,
				Year:         100,
				AltNames:     AltNames{},
				Usages:       nil,
			},
		},
		commonConfig: []CertificateConfig{
			{
				certName: "front-proxy-client",
				descriptor: &CertificateDescriptor{
					CommonName:   "front-proxy-client",
					Organization: nil,
					Year:         100,
					AltNames:     AltNames{},
					Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				},
			},
		},
	}
}

type ClusterCertsArgs struct {
	APIServerAltNames AltNames
	NodeName          string
	NodeIP            net.IP
	DNSDomain         string
}

type KubernetesCertService struct {
	kubeCert       CertificateConfigFamily
	etcdCert       CertificateConfigFamily
	frontProxyCert CertificateConfigFamily
	serviceAccount KeyPairFileGenerator
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

// GenerateAllKubernetesCerts generate all cert.
func GenerateAllKubernetesCerts(certPath, etcdCertPath string, altNames []string, nodeIP net.IP, nodeName, serviceCIRD, DNSDomain string) error {
	// pares args
	data := new(ClusterCertsArgs)
	data.DNSDomain = DNSDomain
	data.APIServerAltNames.IPs = make(map[string]net.IP)
	data.APIServerAltNames.DNSNames = make(map[string]string)
	_, svcSubnet, err := net.ParseCIDR(serviceCIRD)
	if err != nil {
		return errors.Wrapf(err, "unable to parse ServiceSubnet %v", serviceCIRD)
	}
	svcFirstIP, err := utilnet.GetIndexedIP(svcSubnet, 1)
	if err != nil {
		return err
	}
	data.APIServerAltNames.IPs[svcFirstIP.String()] = svcFirstIP

	for _, altName := range altNames {
		ip := net.ParseIP(altName)
		if ip != nil {
			data.APIServerAltNames.IPs[ip.String()] = ip
			continue
		}
		data.APIServerAltNames.DNSNames[altName] = altName
	}

	if nodeIP != nil {
		data.APIServerAltNames.IPs[nodeIP.String()] = nodeIP
	}

	data.NodeIP = nodeIP
	data.NodeName = nodeName

	// get default CertificateConfigFamily
	kubeCert := GetKubeCertificateConfig(certPath, data.APIServerAltNames, data.NodeName, data.DNSDomain)
	etcdCert := GetEtcdCertificateConfig(etcdCertPath, data.NodeName, data.NodeIP)

	// generate all cert.
	certService := KubernetesCertService{
		kubeCert:       kubeCert,
		etcdCert:       etcdCert,
		frontProxyCert: GetFrontProxyCertificateConfig(certPath),
		serviceAccount: NewKeyPairFileGenerator(certPath, "sa"),
	}

	err = certService.GenerateKubeComponentCert()
	if err != nil {
		return err
	}

	err = certService.GenerateServiceAccountKeyPair()
	if err != nil {
		return err
	}

	return nil
}
