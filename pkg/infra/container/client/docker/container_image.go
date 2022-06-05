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

package docker

import (
	"fmt"
	"io"

	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
)

func (p *Provider) DeleteImageResource(imageID string) error {
	_, err := p.DockerClient.ImageRemove(p.Ctx, imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	return err
}

func (p *Provider) PullImage(imageName string) (string, error) {
	// if existed, only set id no need to pull.
	if imageID := p.GetImageIDByName(imageName); imageID != "" {
		return imageID, nil
	}
	out, err := p.DockerClient.ImagePull(p.Ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}

	defer func() {
		_ = out.Close()
	}()

	err = dockerjsonmessage.DisplayJSONMessagesToStream(out, dockerstreams.NewOut(common.StdOut), nil)
	if err != nil && err != io.ErrClosedPipe {
		logrus.Warnf("error occurs in display progressing, err: %s", err)
	}
	logrus.Infof("success to pull docker image: %s", imageName)

	imageID := p.GetImageIDByName(imageName)
	if imageID != "" {
		return imageID, nil
	}

	return "", fmt.Errorf("failed to pull image:%s", imageName)
}

func (p *Provider) GetImageIDByName(name string) string {
	images, err := p.DockerClient.ImageList(p.Ctx, types.ImageListOptions{})
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

func (p *Provider) GetImageResourceByID(id string) (*types.ImageInspect, error) {
	image, _, err := p.DockerClient.ImageInspectWithRaw(p.Ctx, id)
	return &image, err
}
