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
	"os"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

const (
	yes = "y"
	no  = "n"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "delete a cluster",
	Long:    `if provider is BARESERVER will delete kubernetes nodes, or if provider is ALI_CLOUD, will delete all the infra resources`,
	Example: `sealer delete -f /root/.sealer/mycluster/Clusterfile`,
	Run: func(cmd *cobra.Command, args []string) {
		var input string
		for {
			fmt.Printf("Are you sure you want to delete the cluster? Yes [Y/y], No [N/n] : ")
			fmt.Scanln(&input)
			if strings.EqualFold(yes, input) {
				break
			}
			if strings.EqualFold(no, input) {
				fmt.Println("You have canceled to delete the cluster!")
				os.Exit(1)
			}
		}		
		if err := apply.NewApplierFromFile(clusterFile).Delete(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "delete a kubernetes cluster with Clusterfile Annotations")
}
