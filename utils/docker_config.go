package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/alibaba/sealer/common"
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
		return &DockerInfo{Auths: map[string]AuthItem{}}, ioutil.WriteFile(authFile, []byte("{\"auths\":{}}"), common.FileMode0644)
	}

	filebyts, err := ioutil.ReadFile(authFile)
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

	decode, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", err
	}

	spts := strings.Split(string(decode), ":")
	if len(spts) != 2 {
		return "", "", fmt.Errorf("%s auth base64 has problem of format", hostname)
	}

	return spts[0], spts[1], nil
}

func SetDockerConfig(hostname, username, password string) error {
	authFile := common.DefaultRegistryAuthConfigDir()
	authEncode := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	var info *DockerInfo
	var err error
	if !IsFileExist(authFile) {
		if err = os.MkdirAll(filepath.Dir(authFile), common.FileMode0644); err != nil {
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
	if err := ioutil.WriteFile(authFile, data, common.FileMode0644); err != nil {
		return fmt.Errorf("write %s failed,%s", authFile, err)
	}
	return nil
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
