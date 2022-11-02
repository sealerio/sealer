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

package cluster

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/spf13/cobra"
)

var longCertCmdDescription = `This command will add the new domain or IP address in cert to update cluster API server.

sealer has some default domain and IP in the cert process builtin: localhost,outbound IP address and some DNS domain which is strongly related to the apiserver CertSANs configured by kubeadm.yml.

You need to restart your API server manually after using sealer cert. Then, you can using cmd "openssl x509 -noout -text -in apiserver.crt" to check the cert details.
`
var exampleForCertCmd = `
The following command will generate new api server cert and key for all control-plane certificates:

sealer cert --alt-names 39.105.169.253,sealer.cool
`

var altNames []string

func NewCertCmd() *cobra.Command {
	certCmd := &cobra.Command{
		Use:     "cert",
		Short:   "Update Kubernetes API server's cert",
		Args:    cobra.NoArgs,
		Long:    longCertCmdDescription,
		Example: exampleForCertCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(altNames) == 0 {
				return fmt.Errorf("IP address or DNS domain needed for cert Subject Alternative Names")
			}

			workClusterfile := common.GetDefaultClusterfile()
			clusterFileData, err := ioutil.ReadFile(filepath.Clean(workClusterfile))
			if err != nil {
				return err
			}

			cf, err := clusterfile.NewClusterFile(clusterFileData)
			if err != nil {
				return err
			}

			cluster := cf.GetCluster()
			infraDriver, err := infradriver.NewInfraDriver(&cluster, nil)
			if err != nil {
				return err
			}

			certUpdateCmd := fmt.Sprintf("seautil cert update --alt-names %s", strings.Join(altNames, ","))
			// modify new api cert to all master.
			for _, ip := range cluster.GetMasterIPList() {
				err = infraDriver.CmdAsync(ip, certUpdateCmd)
				if err != nil {
					return fmt.Errorf("failed to update cluster api server cert: %v", err)
				}
			}

			//TODO, should wait for apiserver reload completion
			time.Sleep(60 * time.Second)

			return nil
		},
	}

	certCmd.Flags().StringSliceVar(&altNames, "alt-names", []string{}, "add DNS domain or IP in certs, if it is already in the cert subject alternative names list, nothing will be changed")

	return certCmd
}
