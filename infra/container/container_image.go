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

package container

import (
	"fmt"
	"io"
	"os"

	"github.com/alibaba/sealer/logger"
	"github.com/docker/docker/api/types"
)

func (c *DockerProvider) DeleteImageResource(imageID string) error {
	_, err := c.DockerClient.ImageRemove(c.Ctx, imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	return err
}

func (c *DockerProvider) PrepareImageResource() error {
	// if exist, only set id no need to pull
	if imageID := c.GetImageIDByName(c.ImageResource.DefaultName); imageID != "" {
		logger.Info("image %s already exists", c.ImageResource.DefaultName)
		c.ImageResource.ID = imageID
		return nil
	}
	reader, err := c.DockerClient.ImagePull(c.Ctx, c.ImageResource.DefaultName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logger.Fatal(err, "unable to read image pull response")
	}
	imageID := c.GetImageIDByName(c.ImageResource.DefaultName)
	if imageID != "" {
		c.ImageResource.ID = imageID
		return nil
	}

	return fmt.Errorf("failed to pull image:%s", c.ImageResource.DefaultName)
}

func (c *DockerProvider) GetImageIDByName(name string) string {
	images, err := c.DockerClient.ImageList(c.Ctx, types.ImageListOptions{})
	if err != nil {
		return ""
	}
	for _, ima := range images {
		named := ima.RepoTags
		for _, imaName := range named {
			if imaName == name {
				return ima.ID
			}
		}
	}
	return ""
}

func (c *DockerProvider) GetImageResourceByID(id string) (*types.ImageInspect, error) {
	image, _, err := c.DockerClient.ImageInspectWithRaw(c.Ctx, id)
	return &image, err
}
