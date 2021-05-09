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

var imageSrc string

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "read image from tar archive file",
	Long:  `sealer load -i kubernetes.tar.gz`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageFileService().Load(imageSrc); err != nil {
			logger.Error("filed to load image from %s,%v", imageSrc, err)
			os.Exit(1)
		}
		logger.Info("load image from %s success", imageSrc)
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
	loadCmd.Flags().StringVarP(&imageSrc, "input", "i", "", "read image from tar archive file")
	_ = loadCmd.MarkFlagRequired("input")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
