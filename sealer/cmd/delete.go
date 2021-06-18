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
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "delete a cluster",
	Long:    `if provider is BARESERVER will delete kubernetes nodes, or if provider is ALI_CLOUD, will delete all the infra resources`,
	Example: `sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]`,
	Run: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		if !force {
			var yesRx = regexp.MustCompile("^(?:y(?:es)?)$")
			var noRx = regexp.MustCompile("^(?:n(?:o)?)$")
			var input string
			for {
				fmt.Printf("Are you sure you want to delete the cluster? Yes [y/yes], No [n/no] : ")
				fmt.Scanln(&input)
				if yesRx.MatchString(input) {
					break
				}
				if noRx.MatchString(input) {
					fmt.Println("You have canceled to delete the cluster!")
					os.Exit(1)
				}
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
	deleteCmd.Flags().BoolP("force", "", false, "We also can input an --force flag to delete cluster by force")
}
