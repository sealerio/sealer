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
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/utils"
	"github.com/alibaba/sealer/pkg/logger"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type removeImageFlag struct {
	force bool
}

var opts removeImageFlag

// rmiCmd represents the rmi command
var rmiCmd = &cobra.Command{
	Use:     "rmi",
	Short:   "Remove local images by name or ID",
	Example: `sealer rmi registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5`,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRemove(args); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
	ValidArgsFunction: utils.ImageListFuncForCompletion,
}

func runRemove(images []string) error {
	imageService, err := image.NewDeleteImageService(opts.force)
	if err != nil {
		return err
	}

	var errs []string
	for _, img := range images {
		if err := imageService.Delete(img); err != nil {
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
	rmiCmd.Flags().BoolVarP(&opts.force, "force", "f", false, "force removal of the image")
}
