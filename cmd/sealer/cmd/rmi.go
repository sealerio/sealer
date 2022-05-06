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
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/utils"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/platform"
)

type removeImageFlag struct {
	platform string
}

var opts removeImageFlag

// rmiCmd represents the rmi command
var rmiCmd = &cobra.Command{
	Use:     "rmi",
	Short:   "remove local images by name",
	Example: `sealer rmi registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args)
	},
	ValidArgsFunction: utils.ImageListFuncForCompletion,
}

func runRemove(images []string) error {
	imageService, err := image.NewImageService()
	if err != nil {
		return err
	}
	var errs []string
	var targetPlatforms []*v1.Platform

	if opts.platform != "" {
		tp, err := platform.ParsePlatforms(opts.platform)
		if err != nil {
			return err
		}
		targetPlatforms = tp
	}

	for _, img := range images {
		if err := imageService.Delete(img, targetPlatforms); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		msg := strings.Join(errs, "\n")
		return errors.New(msg)
	}

	return nil
}

func init() {
	opts = removeImageFlag{}
	rootCmd.AddCommand(rmiCmd)
	rmiCmd.Flags().StringVar(&opts.platform, "platform", "", "set cloud image platform")
}
