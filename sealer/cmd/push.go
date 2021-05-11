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

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push cloud image to registry",
	Long:  `sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/my-kuberentes-cluster-with-dashboard:latest`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageService().Push(args[0]); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		logger.Info("Push %s success", args[0])
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
