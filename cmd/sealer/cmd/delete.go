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

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"

	"github.com/spf13/cobra"
)

var (
	deleteArgs        *apply.Args
	deleteClusterFile string
	deleteClusterName string
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete an existing cluster",
	Long: `delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying node IP, Clusterfile, or cluster name.`,
	Args: cobra.NoArgs,
	Example: `
delete default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
	sealer delete -c my-cluster [--force]
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}
		if deleteClusterName == "" && deleteClusterFile == "" {
			if !all && deleteArgs.Masters == "" && deleteArgs.Nodes == "" {
				return fmt.Errorf("the delete parameter needs to be set")
			}
			deleteClusterName, err = clusterfile.GetDefaultClusterName()
			if err == clusterfile.ErrClusterNotExist {
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
	deleteArgs = &apply.Args{}
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&deleteArgs.Masters, "masters", "m", "", "reduce Count or IPList to masters")
	deleteCmd.Flags().StringVarP(&deleteArgs.Nodes, "nodes", "n", "", "reduce Count or IPList to nodes")
	deleteCmd.Flags().StringVarP(&deleteClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile Annotations")
	deleteCmd.Flags().StringVarP(&deleteClusterName, "cluster", "c", "", "delete a kubernetes cluster with cluster name")
	deleteCmd.Flags().StringSliceVarP(&deleteArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&kubernetes.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolP("all", "a", false, "this flags is for delete nodes, if this is true, empty all node ip")
}
