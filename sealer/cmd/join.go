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
	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
	"os"
)

var joinArgs *common.RunArgs

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "join node to cluster",
	Example: `
join to default cluster:
	sealer join --master x.x.x.x --node x.x.x.x
join to cluster by cloud provider, just set the number of masters or nodes:
	sealer join --master 2 --node 3
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Lstat(clusterFile); err != nil {
			logger.Error(clusterFile, err)
			os.Exit(1)
		}
		applier := apply.JoinApplierFromArgs(clusterFile, joinArgs)
		if applier == nil {
			os.Exit(1)
		}
		if err := applier.Apply(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	runArgs = &common.RunArgs{}
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVarP(&joinArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&joinArgs.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().StringVarP(&joinArgs.Pk, "pk", "", cert.GetUserHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	joinCmd.Flags().StringVarP(&joinArgs.PkPassword, "pk-passwd", "", "", "set baremetal server  private key password")
	joinCmd.Flags().StringVarP(&joinArgs.Interface, "interface", "i", "", "set default network interface name")
	joinCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", cert.GetUserHomeDir()+"/.sealer/my-cluster/Clusterfile", "apply a kubernetes cluster")
}
