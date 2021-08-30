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
	"os"
	"regexp"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var deleteArgs *common.RunArgs

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a cluster",
	Long:  `if provider is BARESERVER will delete kubernetes nodes or IPList;  if provider is ALI_CLOUD, will delete all the infra resources or count`,
	Example: `
delete to default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete to cluster by cloud provider, just set the number of masters or nodes:
	sealer delete --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
	sealer delete --masters 2 --nodes 3 -f /root/.sealer/specify-cluster/Clusterfile
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
`,
	Run: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		if all && !force {
			var yesRx = regexp.MustCompile("^(?:y(?:es)?)$")
			var noRx = regexp.MustCompile("^(?:n(?:o)?)$")
			var input string
			for {
				fmt.Printf("Are you sure to delete the cluster? Yes [y/yes], No [n/no] : ")
				fmt.Scanln(&input)
				if yesRx.MatchString(input) {
					break
				}
				if noRx.MatchString(input) {
					fmt.Println("You have canceled to delete the cluster!")
					os.Exit(0)
				}
			}
		}
		if deleteArgs.Nodes != "" || deleteArgs.Masters != "" {
			applier := apply.NewScalingApplierFromArgs(clusterFile, deleteArgs)
			if applier == nil {
				os.Exit(1)
			}
			if err := applier.Apply(); err != nil {
				logger.Error(err)
				os.Exit(1)
			}
		} else {
			applier, err := apply.NewApplierFromFile(clusterFile)
			if err != nil {
				logger.Error(err)
				os.Exit(1)
			}
			if err = applier.Delete(); err != nil {
				logger.Error(err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	deleteArgs = &common.RunArgs{}
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&deleteArgs.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteArgs.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "delete a kubernetes cluster with Clusterfile Annotations")
	deleteCmd.Flags().BoolP("force", "", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolP("all", "a", false, "this flags is for delete nodes, if this is true, empty all node ip")
	fmt.Println("hello world")
}
