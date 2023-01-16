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
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/common"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/imageengine/buildah"

	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var umountAll bool

var longUmountCmdDescription = `
umount the cluster image and delete the mount directory
`

var exampleForUmountCmd = `
  sealer alpha umount my-image
  sealer alpha umount ba15e47f5969
  sealer alpha umount --all
`

func NewUmountCmd() *cobra.Command {
	umountCmd := &cobra.Command{
		Use:     "umount",
		Short:   "umount cluster image",
		Long:    longUmountCmdDescription,
		Example: exampleForUmountCmd,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var containerID string
			var imgName string

			if len(args) == 0 && !umountAll {
				return fmt.Errorf("you must input imageName Or imageIp")
			}

			imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			engine, err := buildah.NewBuildahImageEngine(imagecommon.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}

			store := engine.ImageStore()
			containers, err := store.Containers()
			if err != nil {
				return err
			}

			files, err := ioutil.ReadDir(common.DefaultLayerDir)
			if err != nil {
				return err
			}

			// umount all cluster image
			if umountAll {
				for _, c := range containers {
					if err := imageEngine.RemoveContainer(&imagecommon.RemoveContainerOptions{
						ContainerNamesOrIDs: []string{c.ID}},
					); err != nil {
						return err
					}
				}

				for _, file := range files {
					if err := os.RemoveAll(filepath.Join(common.DefaultLayerDir, file.Name())); err != nil {
						return err
					}
				}
				logrus.Infof("umount all cluster image successful")
				return nil
			}

			images, err := store.Images()
			if err != nil {
				return err
			}

			for _, image := range images {
				for _, name := range image.Names {
					if strings.Contains(image.ID, args[0]) {
						imgName = name
					}
				}
			}

			for _, c := range containers {
				if strings.Contains(c.ImageID, args[0]) {
					containerID = c.ID
					break
				}

				id, err := getImageID(images, args[0])
				if err != nil {
					return err
				}
				if c.ImageID == id {
					containerID = c.ID
					imgName = args[0]
				}
			}

			if err := imageEngine.RemoveContainer(&imagecommon.RemoveContainerOptions{
				ContainerNamesOrIDs: []string{containerID}},
			); err != nil {
				return err
			}

			for _, file := range files {
				for _, image := range images {
					if err := removeContainerDir(image, file, imgName); err != nil {
						return err
					}
				}
			}

			logrus.Infof("umount cluster image %s successful", args[0])
			return nil
		},
	}
	umountCmd.Flags().BoolVarP(&umountAll, "all", "a", false, "umount all cluster image directories")
	return umountCmd
}

func getImageID(images []storage.Image, imageName string) (string, error) {
	for _, image := range images {
		for _, n := range image.Names {
			if n == imageName {
				return image.ID, nil
			}
		}
	}
	return "", fmt.Errorf("failed to get container id")
}

func removeContainerDir(image storage.Image, file fs.FileInfo, imageName string) error {
	for _, n := range image.Names {
		if n == imageName && file.Name() == image.ID {
			if err := os.RemoveAll(filepath.Join(common.DefaultLayerDir, file.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
