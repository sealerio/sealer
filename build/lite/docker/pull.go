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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/alibaba/sealer/common"
	dockerstreams "github.com/docker/cli/cli/streams"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Docker struct {
	Username string
	Password string
}

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
	var ImagePullOptions types.ImagePullOptions
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	if d.Username != "" && d.Password != "" {
		authConfig := types.AuthConfig{
			Username: d.Username,
			Password: d.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		ImagePullOptions = types.ImagePullOptions{RegistryAuth: authStr}
	}

	out, err := cli.ImagePull(ctx, image, ImagePullOptions)
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
	return nil
}
