/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
)

var altNames string

// certCmd represents the cert command
var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Update API server cert",
	Long: `Add domain or ip in certs:
    you better to backup your old certs first
	sealer cert --alt-names sealer.cool,10.103.97.2,127.0.0.1,localhost
    uisng "openssl x509 -noout -text -in apiserver.crt" to check the cert
	will update cluster API server cert, you need restart your API server manually after using sealer cert

    For example: add a EIP to cert.
    1. sealer cert --alt-names 39.105.169.253
    2. update the kubeconfig, cp /etc/kubenretes/admin.conf .kube/config
    3. edit .kube/config, set the apiserver address as 39.105.169.253, (don't forget to open the security group port for 6443, if you using public cloud)
    4. kubectl get pod, to check it works or not
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster, err := utils.GetDefaultCluster()
		if err != nil {
			return fmt.Errorf("get default cluster failed, %v", err)
		}
		r, err := runtime.NewDefaultRuntime(cluster, cluster.GetAnnotationsByKey(common.ClusterfileName))
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
