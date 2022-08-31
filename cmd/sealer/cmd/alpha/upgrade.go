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

package alpha

import (
	"github.com/sealerio/sealer/apply"
	"github.com/spf13/cobra"
)

var exampleForUpgradeCmd = `The following command will upgrade the current cluster to kubernetes:v1.19.9
sealer alpha upgrade kubernetes:v1.19.9
`
var longUpgradeCmdDescription = `Sealer upgrade command will upgrade the current cluster to the specified version with the ClusterImage using kubeadm upgrade
`

var upgradeClusterName string

// NewUpgradeCmd implement the sealer upgrade command
func NewUpgradeCmd() *cobra.Command {
	upgradeCmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade specified Kubernetes cluster",
		Long:    longUpgradeCmdDescription,
		Example: exampleForUpgradeCmd,
		Args:    cobra.ExactArgs(1),
		RunE:    upgradeAction,
	}

	upgradeCmd.Flags().StringVarP(&upgradeClusterName, "cluster", "c", "", "the name of cluster")

	return upgradeCmd
}

func upgradeAction(cmd *cobra.Command, args []string) error {
	desiredCluster, err := GetCurrentClusterByName(upgradeClusterName)
	if err != nil {
		return err
	}

	applier, err := apply.NewDefaultApplier(desiredCluster)
	if err != nil {
		return err
	}

	return applier.Upgrade(args[0])
}
