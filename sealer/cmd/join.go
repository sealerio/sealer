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
	"github.com/alibaba/sealer/pkg/clusterfile"
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/common"
)

var clusterName string
var joinArgs *common.RunArgs

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "join node to cluster",
	Args:  cobra.NoArgs,
	Example: `
join to default cluster: merge
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
join to cluster by cloud provider, just set the number of masters or nodes:
	sealer join --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer join --masters 2 --nodes 3 -c my-cluster
`,
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

func init() {
	joinArgs = &common.RunArgs{}
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVarP(&joinArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "submit one cluster name")
}
