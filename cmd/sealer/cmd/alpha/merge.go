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

var (
	mergeImageName string
	mergePlatform  string
)

var exampleForMergeCmd = `Merge mysql,redis and kubernetes image as one ClusterImage named my-image:v1:
	sealer alpha merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0 -t my-image:v1`

var longMergeCmdDescription = `Sealer merge command will merge all layers of source image into one target image`

func NewMergeCmd() *cobra.Command {
	mergeCmd := &cobra.Command{
		Use:     "merge",
		Short:   "Merge multiple images into one",
		Long:    longMergeCmdDescription,
		Example: exampleForMergeCmd,
		Args:    cobra.MinimumNArgs(1),
		RunE:    getMergeFunc,
	}

	mergeCmd.Flags().StringVarP(&mergeImageName, "target-image", "t", "", "target image name")
	mergeCmd.Flags().StringVar(&mergePlatform, "platform", "", "set ClusterImage platform, if not set,keep same platform with runtime")

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
	targetPlatform, err := platform.GetPlatform(mergePlatform)
	if err != nil {
		return err
	}

	if len(targetPlatform) != 1 {
		return fmt.Errorf("merge action only do the same plaform at a time")
	}

	ima := buildRaw(mergeImageName)
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
