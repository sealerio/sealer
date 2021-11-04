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
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/common"
)

var runArgs *common.RunArgs

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a cluster with images and arguments",
	Long:  `sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters [arg] --nodes [arg]`,
	Example: `
create default cluster:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8

create cluster by cloud provider, just set the number of masters or nodes,and default provider is ALI_CLOUD:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters 3 --nodes 3

create cluster by docker container, set the number of masters or nodes, and set provider "CONTAINER":
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters 3 --nodes 3 --provider CONTAINER

create cluster to your baremetal server, appoint the iplist:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
		--nodes 192.168.0.5,192.168.0.6,192.168.0.7
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		applier, err := apply.NewApplierFromArgs(args[0], runArgs)
		if err != nil {
			return err
		}
		return applier.Apply()
	},
}

func init() {
	runArgs = &common.RunArgs{}
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runArgs.Provider, "provider", "", "", "set infra provider, example `ALI_CLOUD`, the local server need ignore this")
	runCmd.Flags().StringVarP(&runArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	runCmd.Flags().StringVarP(&runArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	runCmd.Flags().StringVarP(&runArgs.User, "user", "u", "root", "set baremetal server username")
	runCmd.Flags().StringVarP(&runArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	runCmd.Flags().StringVarP(&runArgs.Pk, "pk", "", cert.GetUserHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	runCmd.Flags().StringVarP(&runArgs.PkPassword, "pk-passwd", "", "", "set baremetal server  private key password")
	runCmd.Flags().StringVarP(&runArgs.PodCidr, "podcidr", "", "", "set default pod CIDR network. example '10.233.0.0/18'")
	runCmd.Flags().StringVarP(&runArgs.SvcCidr, "svccidr", "", "", "set default service CIDR network. example '10.233.64.0/18'")
}
