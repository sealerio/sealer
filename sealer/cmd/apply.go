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
	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"

	"os"
)

var clusterFile string

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:     "apply",
	Short:   "apply a kubernetes cluster",
	Example: `sealer apply -f Clusterfile`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := apply.NewApplierFromFile(clusterFile).Apply(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", "Clusterfile", "apply a kubernetes cluster")
}
