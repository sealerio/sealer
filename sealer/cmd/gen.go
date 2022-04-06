/*
Copyright © 2022 Alibaba Group Holding Ltd.

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

	"github.com/alibaba/sealer/apply/processor"
	"github.com/alibaba/sealer/pkg/cert"
	"github.com/spf13/cobra"
)

var flag *processor.ParserArg

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate a Clusterfile to take over a normal cluster which not deployed by sealer",
	Long: `sealer gen --passwd xxxx --image kubernetes:v1.19.8

The takeover actually is to generate a Clusterfile by kubeconfig.
Sealer will call kubernetes API to get masters and nodes IP info, then generate a Clusterfile.
Also sealer will pull a CloudImage which matches the kubernetes version.

Check generated Clusterfile: 'cat .sealer/<cluster name>/Clusterfile'

The master should has 'node-role.kubernetes.io/master' label.

Then you can use any sealer command to manage the cluster like:

> Upgrade cluster
	sealer upgrade --image kubernetes:v1.22.0

> Scale
	sealer join --node x.x.x.x

> Deploy a CloudImage into the cluster
	sealer run mysql-cluster:5.8`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flag.Passwd == "" || flag.Image == "" {
			return fmt.Errorf("empty password or image name")
		}
		cluster, err := processor.GenerateCluster(flag)
		if err != nil {
			return err
		}
		genProcessor, err := processor.NewGenerateProcessor()
		if err != nil {
			return err
		}
		return processor.NewExecutor(genProcessor).Execute(cluster)
	},
}

func init() {
	flag = &processor.ParserArg{}
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().Uint16Var(&flag.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	genCmd.Flags().StringVar(&flag.Pk, "pk", cert.GetUserHomeDir()+"/.ssh/id_rsa", "set server private key")
	genCmd.Flags().StringVar(&flag.PkPassword, "pk-passwd", "", "set server private key password")
	genCmd.Flags().StringVar(&flag.Image, "image", "", "Set taken over cloud image")
	genCmd.Flags().StringVar(&flag.Name, "name", "default", "Set taken over cluster name")
	genCmd.Flags().StringVar(&flag.Passwd, "passwd", "", "Set taken over ssh passwd")
}
