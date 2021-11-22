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

package cert

import (
	"fmt"

	"github.com/alibaba/sealer/common"
)

// CMD return sealer cert command
func CMD(altNames []string, hostIP, hostName, serviceCIRD, DNSDomain string) string {
	cmd := "sealer cert "
	if hostIP != "" {
		cmd += fmt.Sprintf(" --node-ip %s", hostIP)
	}

	if hostName != "" {
		cmd += fmt.Sprintf(" --node-name %s", hostName)
	}

	if serviceCIRD != "" {
		cmd += fmt.Sprintf(" --service-cidr %s", serviceCIRD)
	}

	if DNSDomain != "" {
		cmd += fmt.Sprintf(" --dns-domain %s", DNSDomain)
	}

	for _, name := range altNames {
		if name != "" {
			cmd += fmt.Sprintf(" --alt-names %s", name)
		}
	}

	return cmd
}

// GenerateCert generate all cert.
func GenerateCert(certPATH, certEtcdPATH string, altNames []string, hostIP, hostName, serviceCIRD, DNSDomain string) error {
	certConfig, err := NewMetaData(certPATH, certEtcdPATH, altNames, serviceCIRD, hostName, hostIP, DNSDomain)
	if err != nil {
		return fmt.Errorf("generator cert config failed %v", err)
	}
	return certConfig.GenerateAll()
}

func GenerateRegistryCert(registryCertPath string, BaseName string) error {
	regCertConfig := Config{
		Path:         registryCertPath,
		BaseName:     BaseName,
		CommonName:   BaseName,
		Organization: []string{common.ExecBinaryFileName},
		Year:         100,
	}
	cert, key, err := NewCaCertAndKey(regCertConfig)
	if err != nil {
		return err
	}
	return WriteCertAndKey(regCertConfig.Path, regCertConfig.BaseName, cert, key)
}
