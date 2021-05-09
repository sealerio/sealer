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

	"github.com/spf13/cobra"
)

var ImageTar string

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "write the image to a file and default tar file name is image id ",
	Long: `sealer save -o [file name] [image name]
examples:
save image by image name:
sealer save -o kubernetes.tar.gz kubernetes:v1.18.3`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageFileService().Save(args[0], ImageTar); err != nil {
			logger.Error("failed to save %v,%v", args[0], err)
		}
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&ImageTar, "output", "o", "", "write the image to a file")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// saveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// saveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
