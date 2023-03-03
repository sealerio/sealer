// Copyright © 2022 Alibaba Group Holding Ltd.
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

package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	osi "github.com/sealerio/sealer/utils/os"

	"github.com/docker/docker/api/types"

	"github.com/sealerio/sealer/common"
)

type Item struct {
	Auth string `json:"auth"`
}

type DockerAuth struct {
	Auths map[string]Item `json:"auths"`
}

func (d *DockerAuth) Get(domain string) (string, string, error) {
	auth := d.Auths[domain].Auth
	if auth == "" {
		return "", "", fmt.Errorf("auth for %s doesn't exist", domain)
	}

	decode, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", err
	}
	i := bytes.IndexRune(decode, ':')

	if i == -1 {
		return "", "", fmt.Errorf("auth base64 has problem of format")
	}

	return string(decode[:i]), string(decode[i+1:]), nil
}

type DockerAuthService struct {
	FilePath    string
	AuthContent DockerAuth `json:"auths"`
}

func (s *DockerAuthService) GetAuthByDomain(domain string) (types.AuthConfig, error) {
	defaultAuthConfig := types.AuthConfig{ServerAddress: domain}

	user, passwd, err := s.AuthContent.Get(domain)
	if err != nil {
		return defaultAuthConfig, err
	}

	return types.AuthConfig{
		Username:      user,
		Password:      passwd,
		ServerAddress: domain,
	}, nil
}

func NewDockerAuthService() (DockerAuthService, error) {
	var (
		authFile = common.DefaultRegistryAuthConfigDir()
		ac       = DockerAuth{Auths: map[string]Item{}}
		das      = DockerAuthService{FilePath: authFile, AuthContent: ac}
	)

	if !osi.IsFileExist(authFile) {
		return das, nil
	}

	content, err := os.ReadFile(filepath.Clean(authFile))
	if err != nil {
		return das, err
	}

	err = json.Unmarshal(content, &ac)
	if err != nil {
		return das, err
	}
	das.AuthContent = ac
	return das, nil
}
