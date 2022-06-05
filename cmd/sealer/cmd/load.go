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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/image"
)

var imageSrc string

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:     "load",
	Short:   "load a CloudImage from a tar file",
	Long:    `Load a CloudImage from a tar archive`,
	Example: `sealer load -i kubernetes.tar`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ifs, err := image.NewImageFileService()
		if err != nil {
			return err
		}
		if err = ifs.Load(imageSrc); err != nil {
			return fmt.Errorf("failed to load image from %s: %v", imageSrc, err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
	loadCmd.Flags().StringVarP(&imageSrc, "input", "i", "", "read image from tar archive file")
	if err := loadCmd.MarkFlagRequired("input"); err != nil {
		logrus.Errorf("failed to init flag: %v", err)
		os.Exit(1)
	}
}
