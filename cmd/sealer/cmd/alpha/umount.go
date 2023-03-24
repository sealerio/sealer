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
	"context"
	"fmt"
	"strings"

	"github.com/containers/buildah"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	imagebuildah "github.com/sealerio/sealer/pkg/imageengine/buildah"
)

var (
	umountAll bool

	longUmountCmdDescription = `
umount the sealer image and delete the mount directory
`
	exampleForUmountCmd = `
  sealer alpha umount imageID
  sealer alpha umount --all
`
)

func NewUmountCmd() *cobra.Command {
	umountCmd := &cobra.Command{
		Use:     "umount",
		Short:   "umount sealer image",
		Long:    longUmountCmdDescription,
		Example: exampleForUmountCmd,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !umountAll {
				return fmt.Errorf("you must input imageName Or containerID")
			}

			umountInfo, err := NewMountService()
			if err != nil {
				return err
			}

			if umountAll {
				if err := umountInfo.UmountAllContainers(); err != nil {
					return err
				}
				logrus.Infof("successful to umount all sealer image")
				return nil
			}

			if err := umountInfo.Umount(args[0]); err != nil {
				return err
			}
			return nil
		},
	}
	umountCmd.Flags().BoolVarP(&umountAll, "all", "a", false, "umount all sealer image directories")
	return umountCmd
}

func (m MountService) Umount(imageID string) error {
	var containerID string
	for _, cid := range m.containers {
		if strings.HasPrefix(cid.ImageID, imageID) {
			containerID = cid.ID
		}
	}

	client, err := imagebuildah.OpenBuilder(context.TODO(), m.store, containerID)
	if err != nil {
		return fmt.Errorf("failed to reading build container %s: %w", containerID, err)
	}

	if err := client.Unmount(); err != nil {
		return fmt.Errorf("failed to unmount container %q: %w", client.Container, err)
	}
	if err := client.Delete(); err != nil {
		return err
	}
	logrus.Infof("umount %s successful", containerID)
	return nil
}

func (m MountService) UmountAllContainers() error {
	clients, err := buildah.OpenAllBuilders(m.store)
	if err != nil {
		return fmt.Errorf("reading build Containers: %w", err)
	}
	for _, client := range clients {
		if client.MountPoint == "" {
			continue
		}

		if err := client.Unmount(); err != nil {
			return fmt.Errorf("failed to unmount container %q: %w", client.Container, err)
		}
		if err := client.Delete(); err != nil {
			return err
		}
	}
	return nil
}
