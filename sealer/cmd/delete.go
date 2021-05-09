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
	"os"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a cluster",
	Long: `sealer delete -f /root/.sealer/mycluster/Clusterfile
if provider is BARESERVER will delete kubernetes nodes, or if provider is ALI_CLOUD, will delete all the infra resources`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := apply.NewApplierFromFile(clusterFile).Delete(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "delete a kubernetes cluster with Clusterfile Annotations")
	_ = deleteCmd.MarkFlagRequired("Clusterfile")
}
