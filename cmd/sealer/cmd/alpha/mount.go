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
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/storage"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/options"
	imagebuildah "github.com/sealerio/sealer/pkg/imageengine/buildah"
	utilsstrings "github.com/sealerio/sealer/utils/strings"
)

var longMountCmdDescription = `
mount the sealer image to '/var/lib/containers/storage/overlay' the directory and check whether the contents of the build image and rootfs are consistent in advance
`

var exampleForMountCmd = `
  sealer alpha mount(show mount list)
  sealer alpha mount my-image
  sealer alpha mount imageID
`

const (
	tableHeaderMountPath = "MOUNT PATH"
	tableHeaderImageID   = "IMAGE ID"
)

type MountService struct {
	table      *tablewriter.Table
	engine     *imagebuildah.Engine
	store      storage.Store
	images     []storage.Image
	containers []storage.Container
	builders   []*buildah.Builder
}

func NewMountCmd() *cobra.Command {
	mountCmd := &cobra.Command{
		Use:     "mount",
		Short:   "mount sealer image",
		Long:    longMountCmdDescription,
		Example: exampleForMountCmd,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mountInfo, err := NewMountService()
			if err != nil {
				return err
			}

			//output mount list
			if len(args) == 0 {
				if err := mountInfo.Show(); err != nil {
					return err
				}
				return nil
			}

			_, err = mountInfo.Mount(args[0])
			if err != nil {
				return err
			}
			return nil
		},
	}
	return mountCmd
}

func NewMountService() (MountService, error) {
	engine, err := imagebuildah.NewBuildahImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return MountService{}, err
	}
	store := engine.ImageStore()
	containers, err := store.Containers()
	if err != nil {
		return MountService{}, err
	}
	images, err := store.Images()
	if err != nil {
		return MountService{}, err
	}

	builders, err := buildah.OpenAllBuilders(store)
	if err != nil {
		return MountService{}, err
	}

	table := tablewriter.NewWriter(common.StdOut)
	table.SetHeader([]string{tableHeaderImageID, tableHeaderMountPath})

	return MountService{
		table:      table,
		engine:     engine,
		store:      store,
		images:     images,
		containers: containers,
		builders:   builders,
	}, nil
}

func (m MountService) Show() error {
	clients, err := buildah.OpenAllBuilders(m.store)
	if err != nil {
		return fmt.Errorf("reading build Containers: %w", err)
	}
	for _, client := range clients {
		mounted, err := client.Mounted()
		if err != nil {
			return err
		}
		for _, container := range m.containers {
			if client.ContainerID == container.ID && mounted {
				imageID := imagebuildah.TruncateID(container.ImageID, true)
				m.table.Append([]string{imageID, client.MountPoint})
			}
		}
	}
	m.table.Render()
	return nil
}

func (m MountService) getMountedImageID(container storage.Container) string {
	for _, image := range m.images {
		if container.ImageID == image.ID {
			return image.ID
		}
	}
	return ""
}

func (m MountService) getImageID(name string) string {
	for _, image := range m.images {
		if strings.HasPrefix(image.ID, name) {
			return image.ID
		}
		for _, n := range image.Names {
			if name == n {
				return image.ID
			}
		}
	}
	return ""
}

func (m MountService) Mount(imageNameOrID string) (string, error) {
	var imageIDList []string

	for _, builder := range m.builders {
		mounted, err := builder.Mounted()
		if err != nil {
			return "", err
		}
		for _, container := range m.containers {
			if builder.ContainerID == container.ID && mounted {
				imageID := m.getMountedImageID(container)
				imageIDList = append(imageIDList, imageID)
			}
		}
	}

	imageID := m.getImageID(imageNameOrID)
	ok := utilsstrings.IsInSlice(imageID, imageIDList)
	if ok {
		logrus.Warnf("this image has already been mounted, please do not repeat the operation")
		return "", nil
	}
	cid, err := m.engine.CreateContainer(&options.FromOptions{
		Image: imageID,
		Quiet: false,
	})
	if err != nil {
		return "", err
	}
	mounts, err := m.engine.Mount(&options.MountOptions{Containers: []string{cid}})
	if err != nil {
		return "", err
	}
	mountPoint := mounts[0].MountPoint
	logrus.Infof("mount sealer image %s to %s successful", imageNameOrID, mountPoint)

	return mountPoint, nil
}
