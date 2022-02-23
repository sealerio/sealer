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
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/pkg/gen"
)

type genFlag struct {
	name   string
	passwd string
	image  string
}

var flag *genFlag

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate a Clusterfile to take over a normal cluster which not deployed by sealer",
	Long: `sealer gen --passwd xxxx --image kubernetes:v1.19.8

The takeover actually is to generate a Clusterfile by kubeconfig.
Sealer will call kubernetes API to get masters and nodes IP info, then generate a Clusterfile.
Also sealer will pull a CloudImage which matchs the kubernetes version.

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
		return gen.GenerateClusterfile(flag.name, flag.passwd, flag.image)
	},
}

func init() {
	flag = &genFlag{}
	rootCmd.AddCommand(genCmd)

	genCmd.Flags().StringVar(&flag.image, "image", "", "Set tackover cloudimage")
	genCmd.Flags().StringVar(&flag.name, "name", "default", "Set tackover cluster name")
	genCmd.Flags().StringVar(&flag.passwd, "passwd", "", "Set tackover ssh passwd")
}
