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
	"regexp"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/apply/v2"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
)

var deleteArgs *common.RunArgs
var deleteClusterFile string
var deleteClusterName string

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a cluster",
	Long:  `if provider is BARESERVER will delete kubernetes nodes or IPList;  if provider is ALI_CLOUD, will delete all the infra resources or count`,
	Args:  cobra.NoArgs,
	Example: `
delete to default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete to cluster by cloud provider, just set the number of masters or nodes:
	sealer delete --masters 2 --nodes 3
specify the Clusterfile(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
	sealer delete --masters 2 --nodes 3 -f /root/.sealer/specify-cluster/Clusterfile
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
	sealer delete --masters 2 --nodes 3 -c specify-cluster
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
	sealer delete -c my-cluster [--force]
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}
		if deleteClusterName == "" && deleteClusterFile == "" {
			if !all && deleteArgs.Masters == "" && deleteArgs.Nodes == "" {
				return fmt.Errorf("the delete parameter needs to be set")
			}
			deleteClusterName, err = utils.GetDefaultClusterName()
			if err == utils.ErrClusterNotExist {
				fmt.Println("Find no exist cluster, skip delete")
				return nil
			}
			if err != nil {
				return err
			}
			deleteClusterFile = common.GetClusterWorkClusterfile(deleteClusterName)
		} else if deleteClusterName != "" && deleteClusterFile != "" {
			tmpClusterfile := common.GetClusterWorkClusterfile(deleteClusterName)
			if tmpClusterfile != deleteClusterFile {
				return fmt.Errorf("arguments error:%s and %s refer to different clusters", deleteClusterFile, tmpClusterfile)
			}
		} else if deleteClusterFile == "" {
			deleteClusterFile = common.GetClusterWorkClusterfile(deleteClusterName)
		}

		if !force {
			var yesRx = regexp.MustCompile("^(?:y(?:es)?)$")
			var noRx = regexp.MustCompile("^(?:n(?:o)?)$")
			var input string
			for {
				fmt.Printf("Are you sure to delete the cluster? Yes [y/yes], No [n/no] : ")
				_, err := fmt.Scanln(&input)
				if err != nil {
					return err
				}
				if yesRx.MatchString(input) {
					break
				}
				if noRx.MatchString(input) {
					fmt.Println("You have canceled to delete the cluster!")
					return nil
				}
			}
		}
		if deleteArgs.Nodes != "" || deleteArgs.Masters != "" {
			applier, err := apply.NewScaleApplierFromArgs(deleteClusterFile, deleteArgs, common.DeleteSubCmd)
			if err != nil {
				return err
			}
			return applier.Apply()
		}

		applier, err := apply.NewApplierFromFile(deleteClusterFile)
		if err != nil {
			return err
		}
		return applier.Delete()
	},
}

func init() {
	deleteArgs = &common.RunArgs{}
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&deleteArgs.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteArgs.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&deleteClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile Annotations")
	deleteCmd.Flags().StringVarP(&deleteClusterName, "cluster", "c", "", "delete a kubernetes cluster with cluster name")
	deleteCmd.Flags().BoolP("force", "", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolP("all", "a", false, "this flags is for delete nodes, if this is true, empty all node ip")
}
