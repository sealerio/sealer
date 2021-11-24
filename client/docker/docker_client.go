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

	"github.com/docker/docker/client"
)

type Docker struct {
	Auth     string
	Username string
	Password string
	cli      *client.Client
	ctx      context.Context
}

func NewDockerClient() (*Docker, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Docker{
		cli: cli,
		ctx: ctx,
	}, nil
}

func (d Docker) GetServerVersion() (string, error) {
	version, err := d.cli.ServerVersion(context.Background())
	if err != nil {
		return "", err
	}
	return version.Version, nil
}
