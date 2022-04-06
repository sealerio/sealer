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
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/platform"

	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image"
)

type saveFlag struct {
	ImageTar string
	Platform string
}

var save saveFlag

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "save image to a tar file",
	Long:  `sealer save -o [output file name] [image name]`,
	Example: `
save kubernetes:v1.19.8 image to kubernetes.tar file:

sealer save -o kubernetes.tar kubernetes:v1.19.8`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imageTar := save.ImageTar
		if imageTar == "" {
			return fmt.Errorf("imagetar cannot be empty")
		}

		dir, file := filepath.Split(imageTar)
		if dir == "" {
			dir = "."
		}
		if file == "" {
			file = fmt.Sprintf("%s.tar", args[0])
		}
		imageTar = filepath.Join(dir, file)
		// only file path like "/tmp" will lose add image tar name,make sure imageTar with full file name.
		if filepath.Ext(imageTar) != ".tar" {
			imageTar = filepath.Join(imageTar, fmt.Sprintf("%s.tar", args[0]))
		}

		ifs, err := image.NewImageFileService()
		if err != nil {
			return err
		}
		named, err := reference.ParseToNamed(args[0])
		if err != nil {
			return err
		}

		var targetPlatforms []*v1.Platform
		if save.Platform != "" {
			tp, err := platform.ParsePlatforms(save.Platform)
			if err != nil {
				return err
			}
			targetPlatforms = tp
		}

		if err = ifs.Save(named.Raw(), imageTar, targetPlatforms); err != nil {
			return fmt.Errorf("failed to save image %s: %v", args[0], err)
		}
		logger.Info("save image %s to %s successfully", args[0], imageTar)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
	save = saveFlag{}
	saveCmd.Flags().StringVarP(&save.ImageTar, "output", "o", "", "write the image to a file")
	saveCmd.Flags().StringVar(&save.Platform, "platform", "", "set cloud image platform")

	if err := saveCmd.MarkFlagRequired("output"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
