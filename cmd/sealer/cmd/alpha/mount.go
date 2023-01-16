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
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/olekukonko/tablewriter"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/imageengine/buildah"
)

var longMountCmdDescription = `
mount the cluster image to '/var/lib/sealer/data/overlay2' the directory and check whether the contents of the build image and rootfs are consistent in advance
`

var exampleForMountCmd = `
  sealer alpha mount(show mount list)
  sealer alpha mount my-image
  sealer alpha mount ba15e47f5969
`

func NewMountCmd() *cobra.Command {
	mountCmd := &cobra.Command{
		Use:     "mount",
		Short:   "mount cluster image",
		Long:    longMountCmdDescription,
		Example: exampleForMountCmd,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				path    string
				imageID string
			)

			engine, err := buildah.NewBuildahImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			store := engine.ImageStore()
			images, err := store.Images()
			if err != nil {
				return err
			}

			//output mount list
			if len(args) == 0 {
				if err := mountList(images); err != nil {
					return err
				}
				return nil
			}

			for _, i := range images {
				for _, name := range i.Names {
					if name == args[0] || strings.Contains(i.ID, args[0]) {
						imageID = i.ID
						path = filepath.Join(common.DefaultLayerDir, imageID)
					}
				}
			}

			imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			if _, err := imageEngine.CreateWorkingContainer(&options.BuildRootfsOptions{
				DestDir:       path,
				ImageNameOrID: args[0],
			}); err != nil {
				return err
			}
			logrus.Infof("mount cluster image %s to %s successful", args[0], path)
			return nil
		},
	}
	return mountCmd
}

func mountList(images []storage.Image) error {
	table := tablewriter.NewWriter(common.StdOut)
	table.SetHeader([]string{imageName, "mountpath"})
	for _, i := range images {
		for _, name := range i.Names {
			err := filepath.Walk(common.DefaultLayerDir, func(path string, f os.FileInfo, err error) error {
				if f.Name() == i.ID {
					table.Append([]string{name, filepath.Join(common.DefaultLayerDir, i.ID)})
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	table.Render()
	return nil
}
