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

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/runtime"
)

var altNames string

// certCmd represents the cert command
var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "update Kubernetes API server's cert",
	Long: `Add domain or ip in certs:
    you had better backup old certs first.
	sealer cert --alt-names sealer.cool,10.103.97.2,127.0.0.1,localhost
    using "openssl x509 -noout -text -in apiserver.crt" to check the cert
	will update cluster API server cert, you need to restart your API server manually after using sealer cert.

    For example: add an EIP to cert.
    1. sealer cert --alt-names 39.105.169.253
    2. update the kubeconfig, cp /etc/kubernetes/admin.conf .kube/config
    3. edit .kube/config, set the apiserver address as 39.105.169.253, (don't forget to open the security group port for 6443, if you using public cloud)
    4. kubectl get pod, to check if it works or not
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster, err := clusterfile.GetDefaultCluster()
		if err != nil {
			return fmt.Errorf("get default cluster failed, %v", err)
		}
		clusterFile, err := clusterfile.NewClusterFile(cluster.GetAnnotationsByKey(common.ClusterfileName))
		if err != nil {
			return err
		}
		r, err := runtime.NewDefaultRuntime(cluster, clusterFile.GetKubeadmConfig())
		if err != nil {
			return fmt.Errorf("get default runtime failed, %v", err)
		}
		return r.UpdateCert(strings.Split(altNames, ","))
	},
}

func init() {
	rootCmd.AddCommand(certCmd)

	certCmd.Flags().StringVar(&altNames, "alt-names", "", "add domain or ip in certs, sealer.cool or 10.103.97.2")
}
