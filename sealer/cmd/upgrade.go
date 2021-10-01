/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/upgrade"

	"github.com/spf13/cobra"
)

var upgradeArgs common.UpgradeArgs

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "upgrade your kubernetes cluster",
	Long:  `sealer upgrade version-you-expect-to --master [args] --node [args] --passwd [args]`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return upgrade.UpgradeCluster(args[0], upgradeArgs)
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	// Here you will define your flags and configuration settings.
	upgradeCmd.Flags().StringVarP(&upgradeArgs.Masters, "master", "m", "", "The masters in the cluster")
	upgradeCmd.Flags().StringVarP(&upgradeArgs.Nodes, "node", "n", "", "The nodes in the cluster")
	upgradeCmd.Flags().StringVarP(&upgradeArgs.Passwd, "passwd", "p", "", "The root's password to log in")
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// upgradeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// upgradeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
