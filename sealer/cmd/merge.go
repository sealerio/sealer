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
	"github.com/alibaba/sealer/build"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var ImageName string

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge multiple images into one",
	Long:  `sealer merge image1:latest image2:latest image3:latest ......`,
	Example: `
merge images:
	sealer merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0  
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var images []string
		for _, v := range strings.Split(args[0], " ") {
			image := strings.TrimSpace(v)
			if image == "" {
				continue
			}
			images = append(images, image)
		}
		if err := build.Merge(ImageName, images); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		logger.Info("images %s is merged to %s!", strings.Join(images, ","), ImageName)
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	rootCmd.Flags().StringVarP(&ImageName, "imageName", "t", "", "target image name")
}
