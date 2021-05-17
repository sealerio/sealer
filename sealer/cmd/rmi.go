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
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"

	"os"

	"github.com/spf13/cobra"
)

// rmiCmd represents the rmi command
var rmiCmd = &cobra.Command{
	Use:     "rmi",
	Short:   "rmi delete local image",
	Example: `sealer rmi registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageService().Delete(args[0]); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmiCmd)
}
