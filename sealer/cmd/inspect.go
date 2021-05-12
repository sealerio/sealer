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
	"fmt"
	"os"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var clusterFilePrint bool

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "print the imageInformation or clusterFile",
	Long: `sealer inspect kubernetes:v1.18.3
	 	   sealer inspect -c kubernetes:v1.18.3`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := image.GetYamlByImage(args[0])
		if file == "" {
			logger.Error("not found image information")
			os.Exit(1)
		}
		if clusterFilePrint {
			cluster := image.GetClusterFileByImage(args[0])
			if cluster == "" {
				logger.Error("not found clusterFile in registry")
				os.Exit(1)
			}
			fmt.Println(cluster)
		} else {
			fmt.Println(file)
		}
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().BoolVarP(&clusterFilePrint, "imageName", "c", false, "print the clusterFile")
}
