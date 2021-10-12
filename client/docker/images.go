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
	"strings"

	"github.com/docker/distribution/reference"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
)

func (d Docker) ImagesPull(images []string) {
	for _, image := range utils.RemoveDuplicate(images) {
		if image == "" {
			continue
		}
		if strings.HasPrefix(image, "#") {
			continue
		}
		if err := d.ImagePull(strings.TrimSpace(image)); err != nil {
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
		err   error
		out   io.ReadCloser
		named reference.Named
	)
	named, err = GetCanonicalImageName(image)
	if err != nil {
		return fmt.Errorf("failed to parse canonical image name %s : %v", image, err)
	}
	opts := GetCanonicalImagePullOptions(named.String())

	out, err = d.cli.ImagePull(d.ctx, named.String(), opts)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	err = dockerjsonmessage.DisplayJSONMessagesToStream(out, dockerstreams.NewOut(common.StdOut), nil)
	if err != nil && err != io.ErrClosedPipe {
		logger.Warn("error occurs in display progressing, err: %s", err)
	}
	logger.Info("success to pull docker image: %s ", image)
	return nil
}

func (d Docker) DockerRmi(imageID string) error {
	if _, err := d.cli.ImageRemove(d.ctx, imageID, types.ImageRemoveOptions{Force: true, PruneChildren: true}); err != nil {
		return err
	}
	return nil
}

func (d Docker) ImagesList() ([]*types.ImageSummary, error) {
	var list []*types.ImageSummary

	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		list = append(list, &image)
	}

	return list, nil
}
