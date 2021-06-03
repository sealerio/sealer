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
	"os"

	"github.com/alibaba/sealer/common"

	"github.com/alibaba/sealer/cert"

	"github.com/alibaba/sealer/apply"

	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var runArgs *common.RunArgs

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a cluster with images and arguments",
	Long:  `sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.7 --masters [arg] --nodes [arg]`,
	Example: `
create default cluster:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.7

create cluster by cloud provider, just set the number of masters or nodes:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.7 --masters 3 --nodes 3

create cluster to your baremetal server, appoint the iplist:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.7 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
		--nodes 192.168.0.5,192.168.0.6,192.168.0.7
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		applier, err := apply.NewApplierFromArgs(args[0], runArgs)
		if err != nil {
			logger.Error(err)
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
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	runCmd.Flags().StringVarP(&runArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	runCmd.Flags().StringVarP(&runArgs.User, "user", "u", "root", "set baremetal server username")
	runCmd.Flags().StringVarP(&runArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	runCmd.Flags().StringVarP(&runArgs.Pk, "pk", "", cert.GetUserHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	runCmd.Flags().StringVarP(&runArgs.PkPassword, "pk-passwd", "", "", "set baremetal server  private key password")
	runCmd.Flags().StringVarP(&runArgs.Interface, "interface", "i", "", "set default network interface name")
	runCmd.Flags().StringVarP(&runArgs.PodCidr, "podcidr", "", "", "set default pod CIDR network. example '192.168.1.0/24'")
	runCmd.Flags().StringVarP(&runArgs.SvcCidr, "svccidr", "s", "", "set default Service CIDR network. example '10.96.0.0/12'")
}
