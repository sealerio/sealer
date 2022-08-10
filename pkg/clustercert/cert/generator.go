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
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/keyutil"
)

const (
	// PrivateKeyBlockType is a possible value for pem.Block.Type.
	PrivateKeyBlockType = "PRIVATE KEY"
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// RSAPrivateKeyBlockType is a possible value for pem.Block.Type.
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	rsaKeySize             = 2048
	duration365d           = time.Hour * 24 * 365
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

	privateKeyPath := PathForKey(k.path, k.name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM :%v", err)
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %s :%v", privateKeyPath, err)
	}

	return nil
}

func (k KeyPairFileGenerator) writePublicKey(key crypto.PublicKey) error {
	if key == nil {
		return errors.New("public key cannot be nil when writing to file")
	}

	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return err
	}

	block := pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}

	publicKeyPath := PathForPublicKey(k.path, k.name)
	if err := keyutil.WriteKey(publicKeyPath, pem.EncodeToMemory(&block)); err != nil {
		return fmt.Errorf("unable to write public key to file %s %v", publicKeyPath, err)
	}

	return nil
}

// AltNames contains the domain names and IP addresses that will be added
// to the API Server's x509 certificate SubAltNames field. The values will
// be passed directly to the x509.Certificate object.
type AltNames struct {
	DNSNames map[string]string
	IPs      map[string]net.IP
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
	config CertificateDescriptor
}

func (m AuthorityCertificateGenerator) Generate() (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(x509.UnknownPublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating CA certificate: %v", err)
	}

	cert, err := m.generateSelfSignedCACert(key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate cert %v", err)
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

func NewAuthorityCertificateGenerator(config CertificateDescriptor) CertificateGenerator {
	return AuthorityCertificateGenerator{
		config: config,
	}
}

type CommonCertificateGenerator struct {
	config CertificateDescriptor
	caCert *x509.Certificate
	caKey  crypto.Signer
}

func (m CommonCertificateGenerator) Generate() (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(x509.UnknownPublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating common certificate: %v", err)
	}

	cert, err := m.generateSignedCert(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate signed cert: %v", err)
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

func NewCommonCertificateGenerator(config CertificateDescriptor, caCert *x509.Certificate, caKey crypto.Signer) (CertificateGenerator, error) {
	if config.CommonName == "" {
		return nil, errors.New("must specify a CommonName for cert")
	}

	if len(config.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	return CommonCertificateGenerator{
		config: config,
		caCert: caCert,
		caKey:  caKey,
	}, nil
}

func PathForCert(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}

func PathForKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.key", name))
}

func PathForPublicKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.pub", name))
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if keyType == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// EncodeCertPEM returns PEM-encoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
