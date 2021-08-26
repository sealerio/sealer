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
	"io/ioutil"
	"os"
	"strings"

	"github.com/alibaba/sealer/cert"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var clusterName string
var joinArgs *common.RunArgs

func getClusterName(sealerPath string) ([]string, error) {
	files, err := ioutil.ReadDir(sealerPath)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	var clusters []string
	for _, f := range files {
		if f.IsDir() {
			clusters = append(clusters, f.Name())
		}
	}
	return clusters, nil
}

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "join node to cluster",
	Example: `
join to default cluster: merge
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
join to cluster by cloud provider, just set the number of masters or nodes:
	sealer join --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer join --masters 2 --nodes 3 -c my-cluster
`,
	Run: func(cmd *cobra.Command, args []string) {
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
		applier := apply.JoinApplierFromArgs(clusterFilePath, joinArgs)
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
	joinArgs = &common.RunArgs{}
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVarP(&joinArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "submit one cluster name")
}
