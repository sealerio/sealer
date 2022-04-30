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
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/apply"
)

var clusterFile string

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:     "apply",
	Short:   "apply a kubernetes cluster",
	Example: `sealer apply -f Clusterfile`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		applier, err := apply.NewApplierFromFile(clusterFile)
		if err != nil {
			return err
		}
		return applier.Apply()
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "apply a kubernetes cluster")
	applyCmd.Flags().BoolVar(&runtime.ForceDelete, "force", false, "We also can input an --force flag to delete cluster by force")
}
