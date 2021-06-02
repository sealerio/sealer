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

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/logger"
)

type ApplyFlag struct {
	ClusterFile string
}

var applyFlag *ApplyFlag

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply a kubernetes cluster",
	Long:  `seautil apply -f cluster.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		applier := apply.NewApplierFromFile(applyFlag.ClusterFile)
		err := applier.Apply()
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	applyFlag = &ApplyFlag{}
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFlag.ClusterFile, "clusterfile", "f", "", "cluster file filepath")
}
