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
	"sort"
	"strings"

	"github.com/alibaba/sealer/pkg/image/reference"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image"
)

const (
	imageID           = "IMAGE ID"
	imageName         = "IMAGE NAME"
	imageCreate       = "CREATE"
	imageSize         = "SIZE"
	imageArch         = "ARCH"
	imageVariant      = "VARIANT"
	timeDefaultFormat = "2006-01-02 15:04:05"
)

var listCmd = &cobra.Command{
	Use:     "images",
	Short:   "list all cluster images",
	Args:    cobra.NoArgs,
	Example: `sealer images`,
	RunE: func(cmd *cobra.Command, args []string) error {

		ims, err := image.NewImageMetadataService()
		if err != nil {
			return err
		}

		imageMetadataMap, err := ims.List()
		if err != nil {
			return err
		}

		var summaries = make(ManifestList, 0, len(imageMetadataMap))

		for name, manifestList := range imageMetadataMap {
			for _, m := range manifestList.Manifests {
				displayName := name
				create := m.CREATED.Format(timeDefaultFormat)
				size := formatSize(m.SIZE)
				named, err := reference.ParseToNamed(name)
				if err != nil {
					return err
				}

				if reference.IsDefaultDomain(named.Domain()) {
					displayName = named.RepoTag()
					splits := strings.Split(displayName, "/")
					if reference.IsDefaultRepo(splits[0]) {
						displayName = splits[1]
					}
				}

				summaries = append(summaries, ManifestDescriptor{
					imageName:    displayName,
					imageID:      m.ID,
					imageArch:    m.Platform.Architecture,
					imageVariant: m.Platform.Variant,
					imageCreate:  create,
					imageSize:    size})
			}
		}

		sort.Sort(sort.Reverse(summaries))

		table := tablewriter.NewWriter(common.StdOut)
		table.SetHeader([]string{imageName, imageID, imageArch, imageVariant, imageCreate, imageSize})

		for _, md := range summaries {
			table.Append([]string{md.imageName, md.imageID, md.imageArch, md.imageVariant, md.imageCreate, md.imageSize})
		}

		table.Render()
		return nil
	},
}

type ManifestDescriptor struct {
	imageName, imageID, imageArch, imageVariant, imageCreate, imageSize string
}

type ManifestList []ManifestDescriptor

func (r ManifestList) Len() int           { return len(r) }
func (r ManifestList) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ManifestList) Less(i, j int) bool { return r[i].imageCreate < r[j].imageCreate }

func init() {
	rootCmd.AddCommand(listCmd)
}

func formatSize(size int64) (Size string) {
	if size < 1024 {
		Size = fmt.Sprintf("%.2fB", float64(size)/float64(1))
	} else if size < (1024 * 1024) {
		Size = fmt.Sprintf("%.2fKB", float64(size)/float64(1024))
	} else if size < (1024 * 1024 * 1024) {
		Size = fmt.Sprintf("%.2fMB", float64(size)/float64(1024*1024))
	} else {
		Size = fmt.Sprintf("%.2fGB", float64(size)/float64(1024*1024*1024))
	}
	return
}
