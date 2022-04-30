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

package command

import (
	"github.com/sealerio/sealer/utils"
)

type Interface interface {
	Exec() (string, error)
}

type SimpleCommand struct {
	Command string
}

func (c *SimpleCommand) Exec() (string, error) {
	output, err := utils.RunSimpleCmd(c.Command)
	return output, err
}

func NewSimpleCommand(cmd string) Interface {
	return &SimpleCommand{
		Command: cmd,
	}
}

type CopyCommand struct {
	Src string
	Dst string
}

// COPY dashboard-chart .
// COPY [src] [dst]
// copy files to /var/lib/sealer/[image hash]/[layer hash]
func (c *CopyCommand) Exec() (string, error) {
	output, err := utils.CmdOutput("cp", "-r", c.Src, c.Dst)
	if err != nil {
		return "", err
	}
	return string(output), err
}

func NewCopyCommand(src, dst string) Interface {
	return &CopyCommand{
		Src: src,
		Dst: dst,
	}
}
