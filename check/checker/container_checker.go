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

package checker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/client"
)

type ContainerChecker struct {
	SealerContainer bool
}

func (c ContainerChecker) Check() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to init container client %s", err)
	}
	v, err := cli.ServerVersion(ctx)
	if err != nil {
		// container is not installed
		return nil
	}
	c.SealerContainer = strings.HasSuffix(v.Version, "-sealer")
	if !c.SealerContainer {
		return fmt.Errorf("container check failed,we can only use sealer container but we found the version： %s", v.Version)
	}
	return nil
}

func NewContainerChecker() Checker {
	return &ContainerChecker{
		SealerContainer: false,
	}
}
