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

	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/pkg/image/store"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image"
)

var ImageTar string

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "save image",
	Long:  `save image to a file `,
	Example: `
sealer save -o [output file name] [image name]
save kubernetes:v1.19.8 image to kubernetes.tar file:
sealer save -o kubernetes.tar kubernetes:v1.19.8`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ifs, err := image.NewImageFileService()
		if err != nil {
			return err
		}
		named, err := reference.ParseToNamed(args[0])
		if err != nil {
			return err
		}
		imageStore, err := store.NewDefaultImageStore()
		if err != nil {
			return err
		}
		image, err := imageStore.GetByName(named.Raw())
		if err != nil {
			return err
		}
		fmt.Printf("%+v\n", image)
		if err = ifs.Save(image, ImageTar); err != nil {
			return fmt.Errorf("failed to save image %s: %v", args[0], err)
		}
		logger.Info("save image %s to %s successfully", args[0], ImageTar)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&ImageTar, "output", "o", "", "write the image to a file")
	if err := saveCmd.MarkFlagRequired("output"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
