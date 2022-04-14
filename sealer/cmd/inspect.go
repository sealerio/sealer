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

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/platform"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/pkg/image"
)

var inspectPlatformFlag string

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "print the image information or clusterFile",
	Long:  `sealer inspect ${image id} to print image information`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPlatforms []*v1.Platform
		if inspectPlatformFlag != "" {
			tp, err := platform.ParsePlatforms(inspectPlatformFlag)
			if err != nil {
				return err
			}
			targetPlatforms = tp
		}
		file, err := image.GetImageDetails(args[0], targetPlatforms)
		if err != nil {
			return fmt.Errorf("failed to find information by image %s: %v", args[0], err)
		}
		fmt.Println(file)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVar(&inspectPlatformFlag, "platform", "", "set cloud image platform")
}
