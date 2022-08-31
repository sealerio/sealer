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
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
)

var clusterName string
var joinArgs *apply.Args

var exampleForJoinCmd = `
join default cluster:
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
`

func NewJoinCmd() *cobra.Command {
	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "join new master or worker node to specified cluster",
		// TODO: add long description.
		Long:    "",
		Args:    cobra.NoArgs,
		Example: exampleForJoinCmd,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterName == "" {
				cn, err := clusterfile.GetDefaultClusterName()
				if err != nil {
					return err
				}
				clusterName = cn
			}
			path := common.GetClusterWorkClusterfile(clusterName)
			applier, err := apply.NewScaleApplierFromArgs(path, joinArgs, common.JoinSubCmd)
			if err != nil {
				return err
			}
			return applier.Apply()
		},
	}
	joinArgs = &apply.Args{}
	joinCmd.Flags().StringVarP(&joinArgs.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().Uint16Var(&joinArgs.Port, "port", 22, "set the sshd service port number for the server (default port: 22)")
	joinCmd.Flags().StringVar(&joinArgs.Pk, "pk", common.GetHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	joinCmd.Flags().StringVar(&joinArgs.PkPassword, "pk-passwd", "", "set baremetal server private key password")
	joinCmd.Flags().StringSliceVarP(&joinArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	joinCmd.Flags().StringVarP(&joinArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "specify the name of cluster")
	return joinCmd
}
