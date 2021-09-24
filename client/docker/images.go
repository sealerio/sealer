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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/utils"
	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"

	"github.com/alibaba/sealer/logger"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"io"
)

func (d Docker) ImagesPull(images []string) {
	for _, image := range utils.RemoveDuplicate(images) {
		if image == "" {
			continue
		}
		if err := d.ImagePull(image); err != nil {
			logger.Warn(fmt.Sprintf("Image %s pull failed: %v", image, err))
		}
	}
}

func (d Docker) ImagesPullByImageListFile(fileName string) {
	data, err := utils.ReadLines(fileName)
	if err != nil {
		logger.Error(fmt.Sprintf("Read image list failed: %v", err))
	}
	d.ImagesPull(data)
}

func (d Docker) ImagesPullByList(images []string) {
	d.ImagesPull(images)
}

func (d Docker) ImagePull(image string) error {
	var (
		named       reference.Named
		err         error
		authConfig  types.AuthConfig
		out         io.ReadCloser
		encodedJSON []byte
		authStr     string
	)
	named, err = reference.ParseToNamed(image)
	if err != nil {
		logger.Warn("image information parsing failed: %v", err)
		return err
	}
	var ImagePullOptions types.ImagePullOptions

	authConfig, err = utils.GetDockerAuthInfoFromDocker(named.Domain())
	if err == nil {
		encodedJSON, err = json.Marshal(authConfig)
		if err != nil {
			logger.Warn("authConfig encodedJSON failed: %v", err)
		} else {
			authStr = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}

	ImagePullOptions = types.ImagePullOptions{RegistryAuth: authStr}
	out, err = d.cli.ImagePull(d.ctx, image, ImagePullOptions)
	if err != nil {
		logger.Warn("Image pull failed: %v", err)
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	err = dockerjsonmessage.DisplayJSONMessagesToStream(out, dockerstreams.NewOut(common.StdOut), nil)
	if err != nil && err != io.ErrClosedPipe {
		logger.Warn("error occurs in display progressing, err: %s", err)
	}
	return nil
}

func (d Docker) DockerRmi(imageID string) error {
	if _, err := d.cli.ImageRemove(d.ctx, imageID, types.ImageRemoveOptions{Force: true, PruneChildren: true}); err != nil {
		return err
	}
	return nil
}

func (d Docker) ImagesList() ([]*types.ImageSummary, error) {
	var List []*types.ImageSummary
	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		List = append(List, &image)
	}
	return List, nil
}
