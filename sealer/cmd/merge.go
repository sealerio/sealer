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
	"strings"

	"github.com/alibaba/sealer/image"

	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

var ImageName string

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge multiple images into one",
	Long:  `sealer merge image1:latest image2:latest image3:latest ......`,
	Example: `
merge images:
	sealer merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0 -t new:0.1.0
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var images []string
		for _, v := range args {
			image := strings.TrimSpace(v)
			if image == "" {
				continue
			}
			images = append(images, image)
		}
		if ImageName == "" {
			ImageName = "merged:latest"
		}
		if len(strings.Split(ImageName, ":")) == 1 {
			ImageName = ImageName + ":latest"
		}
		if err := image.Merge(ImageName, images); err != nil {
			return err
		}
		logger.Info("images %s is merged to %s!", strings.Join(images, ","), ImageName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringVarP(&ImageName, "image-name", "t", "", "target image name")
}
