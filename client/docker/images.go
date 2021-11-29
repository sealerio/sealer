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

func (d Docker) ImagesCacheToRegistry(images []string) error {
	version, err := d.cli.ServerVersion(d.ctx)
	if err != nil {
		return err
	}
	if strings.Contains(version.Version, `sealer`) {
		return d.ImagesPull(images)
	}
	// need a extra push step
	if err := d.ImagesPull(images); err != nil {
		return err
	}
	return d.ImagesPush(images)
}
func (d Docker) ImagesPush(images []string) error {
	for _, image := range utils.RemoveDuplicate(images) {
		if image == "" {
			continue
		}
		if strings.HasPrefix(image, "#") {
			continue
		}
		if err := d.ImagePush(strings.TrimSpace(image)); err != nil {
			return fmt.Errorf("image %s push failed: %v", image, err)
		}
	}
	return nil
}
func (d Docker) ImagePush(image string) error {
	var (
		err          error
		out          io.ReadCloser
		named        reference.Named
		registryName = `sea.hub:5000`
		TagImageName string
	)
	named, err = GetCanonicalImageName(image)
	if err != nil {
		return fmt.Errorf("failed to parse canonical image name %s : %v", image, err)
	}
	TagImageName = strings.Replace(named.String(), named.String()[0:strings.Index(named.String(), `/`)], registryName, 1)
	opts := types.ImagePushOptions(GetCanonicalImagePullOptions(named.String()))
	if opts.RegistryAuth == "" {
		opts.RegistryAuth = "sea.hub" //must not empty
	}
	err = d.cli.ImageTag(d.ctx, named.String(), TagImageName)
	if err != nil {
		return err
	}
	out, err = d.cli.ImagePush(d.ctx, TagImageName, opts)
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
	logger.Info("success to push docker image: %s ", image)
	return nil
}
func (d Docker) ImagesPull(images []string) error {
	for _, image := range utils.RemoveDuplicate(images) {
		if image == "" {
			continue
		}
		if strings.HasPrefix(image, "#") {
			continue
		}
		if err := d.ImagePull(strings.TrimSpace(image)); err != nil {
			return fmt.Errorf("image %s pull failed: %v", image, err)
		}
	}
	return nil
}

// func (d Docker) ImagesPullByImageListFile(fileName string) error {
// 	data, err := utils.ReadLines(fileName)
// 	if err != nil {
// 		logger.Error(fmt.Sprintf("Read image list failed: %v", err))
// 	}
// 	return d.ImagesPull(data)
// }

// func (d Docker) ImagesPullByList(images []string) error {
// 	return d.ImagesPull(images)
// }

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
