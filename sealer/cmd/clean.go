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
	"strings"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/cert"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var cleanOpts *common.RunOpts
var cleanArgs *common.RunArgs

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "clean node to cluster",
	Example: `
clean to default cluster: merge
	sealer clean --masters x.x.x.x --nodes x.x.x.x
	sealer clean --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
	sealer clean --all -f [--force]
clean to cluster by cloud provider, just set the number of masters or nodes:
	sealer clean --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
	sealer clean --masters 2 --nodes 3 -c my-cluster
`,
	Run: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		if all && !force {
			var yesRx = regexp.MustCompile("^(?:y(?:es)?)$")
			var noRx = regexp.MustCompile("^(?:n(?:o)?)$")
			var input string
			for {
				fmt.Printf("Are you sure to clean the cluster? Yes [y/yes], No [n/no] : ")
				fmt.Scanln(&input)
				if yesRx.MatchString(input) {
					break
				}
				if noRx.MatchString(input) {
					fmt.Println("You have canceled to clean the cluster!")
					os.Exit(0)
				}
			}
		}

		sealerPath := fmt.Sprintf("%s/.sealer", cert.GetUserHomeDir())
		if clusterName == "" {
			files, err := getClusterName(sealerPath)
			if err != nil {
				logger.Error(err)
				os.Exit(1)
			}
			if len(files) == 1 {
				clusterName = files[0]
			} else if len(files) > 1 {
				logger.Error("Select a cluster through the -c parameter:", strings.Join(files, ","))
				os.Exit(1)
			} else {
				logger.Error("Existing cluster not found！")
				os.Exit(1)
			}
		}

		clusterFilePath := fmt.Sprintf("%s/%s/Clusterfile", sealerPath, clusterName)
		if _, err := os.Lstat(clusterFilePath); err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		applier := apply.NewCleanApplierFromArgs(clusterFilePath, cleanArgs, cleanOpts)

		if applier == nil {
			os.Exit(1)
		}
		if err := applier.Apply(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	cleanArgs = &common.RunArgs{}
	cleanOpts = &common.RunOpts{}
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().StringVarP(&cleanArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	cleanCmd.Flags().StringVarP(&cleanArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	cleanCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "submit one cluster name")
	cleanCmd.Flags().BoolVarP(&cleanOpts.All, "all", "a", false, "if this is true, empty all node ip")
	cleanCmd.Flags().BoolVarP(&cleanOpts.Force, "force", "f", false, "if this is true, will no prompt")
}
