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

package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/logger"
	"github.com/cavaliergopher/grab/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type webFileCollector struct {
}

func (w webFileCollector) Send(buildContext, src, savePath string) error {
	client := grab.NewClient()
	i := strings.LastIndexByte(src, '/')
	req, err := grab.NewRequest(filepath.Join(savePath, src[i+1:]), src)
	if err != nil {
		return err
	}
	//todo add progress message stdout same with docker pull.
	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		return err
	}
	return nil
}

func NewWebFileCollector() Collector {
	return webFileCollector{}
}

type gitCollector struct {
}

func (g gitCollector) Send(buildContext, src, savePath string) error {
	co := &git.CloneOptions{
		URL:               src,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          os.Stdout,
	}

	if strings.HasPrefix(src, "git@") {
		privateKeyFile := os.Getenv("HOME") + "/.ssh/id_rsa"
		_, err := os.Stat(privateKeyFile)
		if err != nil {
			logger.Warn("read file %s failed %s\n", privateKeyFile, err.Error())
			return err
		}

		publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyFile, "")
		if err != nil {
			logger.Warn("generate public keys failed: %s\n", err.Error())
			return err
		}
		co.Auth = publicKeys
	}
	_, err := git.PlainClone(savePath, false, co)

	if err != nil {
		return err
	}
	return nil
}

func NewGitCollector() Collector {
	return gitCollector{}
}
