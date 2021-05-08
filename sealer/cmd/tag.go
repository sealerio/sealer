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
	"os"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "tag IMAGE[:TAG] TARGET_IMAGE[:TAG]",
	Long:  `sealer tag sealer/cloudrootfs:v1.16.9-alpha.6 registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.5`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			logger.Error("Enter imageName and target imageName")
			os.Exit(1)
		}
		err := image.NewImageMetadataService().Tag(args[0], args[1])
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}
