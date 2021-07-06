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
	"os"

	"github.com/alibaba/sealer/common"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	imageID   = "IMAGE ID"
	imageName = "IMAGE NAME"
)

var listCmd = &cobra.Command{
	Use:     "images",
	Short:   "list all cluster images",
	Example: `sealer images`,
	Run: func(cmd *cobra.Command, args []string) {
		imageMetadataList, err := image.NewImageMetadataService().List()
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		table := tablewriter.NewWriter(common.StdOut)
		table.SetHeader([]string{imageID, imageName})
		for _, image := range imageMetadataList {
			table.Append([]string{image.ID, image.Name})
		}
		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
