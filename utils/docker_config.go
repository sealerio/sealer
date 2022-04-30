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

package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"

	"github.com/sealerio/sealer/common"
)

type AuthItem struct {
	Auth string `json:"auth"`
}

type DockerInfo struct {
	Auths map[string]AuthItem `json:"auths"`
}

func DockerConfig() (*DockerInfo, error) {
	authFile := common.DefaultRegistryAuthConfigDir()
	if !IsFileExist(authFile) {
		if err := os.MkdirAll(filepath.Dir(authFile), common.FileMode0755); err != nil {
			return nil, err
		}
		return &DockerInfo{Auths: map[string]AuthItem{}}, AtomicWriteFile(authFile, []byte("{\"auths\":{}}"), common.FileMode0644)
	}

	filebyts, err := ioutil.ReadFile(filepath.Clean(authFile))
	if err != nil {
		return nil, err
	}

	dockerInfo := &DockerInfo{}
	err = json.Unmarshal(filebyts, dockerInfo)
	if err != nil {
		return nil, err
	}

	return dockerInfo, nil
}

func (d DockerInfo) LocalDockerAuth(hostname string) string {
	return d.Auths[hostname].Auth
}

func (d DockerInfo) DecodeDockerAuth(hostname string) (string, string, error) {
	auth := d.LocalDockerAuth(hostname)
	if auth == "" {
		return "", "", fmt.Errorf("auth for %s doesn't exist", hostname)
	}

	return DecodeAuth(auth)
}

func DecodeAuth(auth string) (string, string, error) {
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

func SetDockerConfig(hostname, username, password string) error {
	authFile := common.DefaultRegistryAuthConfigDir()
	authEncode := EncodeAuth(username, password)
	var info *DockerInfo
	var err error
	if !IsFileExist(authFile) {
		if err = os.MkdirAll(filepath.Dir(authFile), common.FileMode0755); err != nil {
			return err
		}
		info = &DockerInfo{Auths: map[string]AuthItem{}}
	} else {
		info, err = DockerConfig()
		if err != nil {
			return err
		}
	}
	info.Auths[hostname] = AuthItem{Auth: authEncode}
	data, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return err
	}
	if err := AtomicWriteFile(authFile, data, common.FileMode0644); err != nil {
		return fmt.Errorf("write %s failed,%s", authFile, err)
	}
	return nil
}

func EncodeAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
}

func GetDockerAuthInfoFromDocker(domain string) (types.AuthConfig, error) {
	var (
		dockerInfo        *DockerInfo
		err               error
		username, passwd  string
		defaultAuthConfig = types.AuthConfig{ServerAddress: domain}
	)

	dockerInfo, err = DockerConfig()
	if err != nil {
		return defaultAuthConfig, err
	}

	username, passwd, err = dockerInfo.DecodeDockerAuth(domain)
	if err != nil {
		return defaultAuthConfig, err
	}

	return types.AuthConfig{
		Username:      username,
		Password:      passwd,
		ServerAddress: domain,
	}, nil
}
