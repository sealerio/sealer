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
	cluster_runtime "github.com/sealerio/sealer/pkg/cluster-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"net"

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"

	"github.com/spf13/cobra"
)

var (
	deleteArgs        *apply.Args
	deleteClusterFile string
	deleteClusterName string
	mastersToDelete   []net.IP
	workersToDelete   []net.IP
	deleteAll         bool
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
		var cf clusterfile.Interface
		if clusterFile != "" {
			var err error
			cf, err = clusterfile.NewClusterFile(clusterFile)
			if err != nil {
				return err
			}
		}

		cluster := cf.GetCluster()
		infraDriver, err := infradriver.NewInfraDriver(&cluster)
		if err != nil {
			return err
		}

		installer, err := cluster_runtime.NewInstaller(infraDriver, &cluster)
		if err != nil {
			return err
		}

		if deleteAll {
			if err = installer.UnInstall(); err != nil {
				return err
			}
		} else {
			_, _, err = installer.ScaleDown(mastersToDelete, workersToDelete)
			if err != nil {
				return err
			}
		}

		//TODO remove files from deleted hosts

		if deleteAll {
			//TODO umount image
		}

		return nil
	},
}

func init() {
	deleteArgs = &apply.Args{}
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().IPSliceVarP(&mastersToDelete, "masters", "m", nil, "reduce Count or IPList to masters")
	deleteCmd.Flags().IPSliceVarP(&workersToDelete, "nodes", "n", nil, "reduce Count or IPList to nodes")

	deleteCmd.Flags().StringVarP(&deleteClusterFile, "Clusterfile", "f", "", "delete a kubernetes cluster with Clusterfile Annotations")
	deleteCmd.Flags().StringVarP(&deleteClusterName, "cluster", "c", "", "delete a kubernetes cluster with cluster name")
	deleteCmd.Flags().StringSliceVarP(&deleteArgs.CustomEnv, "env", "e", []string{}, "set custom environment variables")
	deleteCmd.Flags().BoolVar(&kubernetes.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
	deleteCmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "this flags is for delete nodes, if this is true, empty all node ip")
}
