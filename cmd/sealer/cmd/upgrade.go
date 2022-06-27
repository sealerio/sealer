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

	"github.com/sealerio/sealer/apply"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"

	"github.com/spf13/cobra"
)

var upgradeClusterName string

const (
	clusterfilepath = `%s/.sealer/%s/Clusterfile`
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Short:   "upgrade specified Kubernetes cluster",
	Long:    `sealer upgrade imagename --cluster clustername`,
	Example: `sealer upgrade kubernetes:v1.19.9 --cluster my-cluster`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		//get cluster name
		if upgradeClusterName == "" {
			upgradeClusterName, err = clusterfile.GetDefaultClusterName()
			if err != nil {
				return err
			}
		}
		//get Clusterfile
		userHome, _ := os.UserHomeDir()
		var filepath = fmt.Sprintf(clusterfilepath, userHome, upgradeClusterName)
		desiredCluster, err := clusterfile.GetClusterFromFile(filepath)
		if err != nil {
			return err
		}
		applier, err := apply.NewDefaultApplier(desiredCluster, common.UpgradeSubCmd, nil)
		if err != nil {
			return err
		}
		return applier.Upgrade(args[0])
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().StringVarP(&upgradeClusterName, "cluster", "c", "", "the name of cluster")
}
