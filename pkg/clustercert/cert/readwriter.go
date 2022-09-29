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
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/pkg/errors"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

// CertificateFileManger Asymmetric encryption, like ca.crt and ca.key
type CertificateFileManger struct {
	certName string
	certPath string
}

func (c CertificateFileManger) Write(cert *x509.Certificate, key crypto.Signer) error {
	err := c.writeCert(cert)
	if err != nil {
		return err
	}

	err = c.writeKey(key)
	if err != nil {
		return err
	}

	return nil
}

func (c CertificateFileManger) writeKey(key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := PathForKey(c.certPath, c.certName)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM %v", err)
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %s %v", privateKeyPath, err)
	}

	return nil
}

func (c CertificateFileManger) writeCert(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	certificatePath := PathForCert(c.certPath, c.certName)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return fmt.Errorf("unable to write certificate to file %s %v", certificatePath, err)
	}

	return nil
}

func (c CertificateFileManger) Read() (cert *x509.Certificate, key crypto.Signer, err error) {
	key, err = c.readKey()
	if err != nil {
		return
	}

	cert, err = c.readCert()
	if err != nil {
		return
	}

	return
}

func (c CertificateFileManger) readKey() (crypto.Signer, error) {
	// Parse the private key from a file
	privateKey, err := keyutil.PrivateKeyFromFile(PathForKey(c.certPath, c.certName))
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

func (c CertificateFileManger) readCert() (cert *x509.Certificate, err error) {
	certs, err := certutil.CertsFromFile(PathForCert(c.certPath, c.certName))
	if err != nil {
		return nil, err
	}
	return certs[0], nil
}

func NewCertificateFileManger(certPath string, certName string) CertificateFileManger {
	return CertificateFileManger{
		certName: certName,
		certPath: certPath,
	}
}
