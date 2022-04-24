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

	"github.com/alibaba/sealer/pkg/clusterfile"

	"github.com/alibaba/sealer/apply"
	"github.com/spf13/cobra"
)

var uninstallClusterName string

// uninstallCmd represents the upgrade command
var uninstallCmd = &cobra.Command{
	Use:     "uninstall",
	Short:   "uninstall your app installed by sealer",
	Long:    `sealer uninstall imagename --cluster clustername`,
	Example: `sealer uninstall dashboard:v1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		//get cluster name
		if uninstallClusterName == "" {
			uninstallClusterName, err = clusterfile.GetDefaultClusterName()
			if err != nil {
				return err
			}
		}
		//get Clusterfile
		userHome, _ := os.UserHomeDir()

		desiredCluster, err := clusterfile.GetClusterFromFile(fmt.Sprintf(clusterfilepath, userHome, uninstallClusterName))
		if err != nil {
			return err
		}

		applier, err := apply.NewApplier(desiredCluster)
		if err != nil {
			return err
		}
		return applier.Uninstall(args[0])
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().StringVarP(&uninstallClusterName, "cluster", "c", "", "The name of your cluster to uninstall")
}
