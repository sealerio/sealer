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

package alpha

import (
	"fmt"
	"strings"

	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/utils/platform"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mergeFlag struct {
	ImageName string
	Platform  string
}

var mf *mergeFlag

func NewMergeCmd() *cobra.Command {
	var mergeCmd = &cobra.Command{
		Use:   "merge",
		Short: "merge multiple images into one",
		Long:  `sealer merge image1:latest image2:latest image3:latest ......`,
		Example: `
merge images:
	sealer merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0 -t new:0.1.0
`,
		Args: cobra.MinimumNArgs(1),
		RunE: getMergeFunc,
	}
	mf = &mergeFlag{}
	mergeCmd.Flags().StringVarP(&mf.ImageName, "target-image", "t", "", "target image name")
	mergeCmd.Flags().StringVar(&mf.Platform, "platform", "", "set ClusterImage platform, if not set,keep same platform with runtime")

	if err := mergeCmd.MarkFlagRequired("target-image"); err != nil {
		logrus.Errorf("failed to init flag target image: %v", err)
	}
	return mergeCmd
}

func getMergeFunc(cmd *cobra.Command, args []string) error {
	var images []string
	for _, v := range args {
		imageName := strings.TrimSpace(v)
		if imageName == "" {
			continue
		}
		images = append(images, imageName)
	}
	targetPlatform, err := platform.GetPlatform(mf.Platform)
	if err != nil {
		return err
	}

	if len(targetPlatform) != 1 {
		return fmt.Errorf("merge action only do the same plaform at a time")
	}

	ima := buildRaw(mf.ImageName)
	if err := image.Merge(ima, images, targetPlatform[0]); err != nil {
		return err
	}
	logrus.Infof("images %s is merged to %s", strings.Join(images, ","), ima)
	return nil
}

func buildRaw(name string) string {
	defaultTag := "latest"
	i := strings.LastIndexByte(name, ':')
	if i == -1 {
		return name + ":" + defaultTag
	}
	if i > strings.LastIndexByte(name, '/') {
		return name
	}
	return name + ":" + defaultTag
}

