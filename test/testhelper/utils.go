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

package testhelper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"

	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/gomega"
	"k8s.io/client-go/util/homedir"

	"sigs.k8s.io/yaml"
)

func GetPwd() string {
	pwd, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pwd
}

func CreateTempFile() string {
	dir := os.TempDir()
	file, err := ioutil.TempFile(dir, "tmpfile")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer func() {
		err := file.Close()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}()
	return file.Name()
}

func RemoveTempFile(file string) {
	err := os.Remove(file)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func GetUserHomeDir() string {
	if home := homedir.HomeDir(); home != "" {
		return home
	}
	return ""
}

func WriteFile(fileName string, content []byte) error {
	dir := filepath.Dir(fileName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, settings.FileMode0755); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(fileName, content, settings.FileMode0644); err != nil {
		return err
	}
	return nil
}

func GetUsedClusterFilePath(clusterName string) string {
	if home := homedir.HomeDir(); home != "" {
		return fmt.Sprintf("%s/.sealer/%s/Clusterfile", home, clusterName)
	}
	return ""
}

type SSHClient struct {
	RemoteHostIP string
	SSH          ssh.Interface
}

func NewSSHClientByCluster(usedCluster *v1.Cluster) *SSHClient {
	sshClient, err := ssh.NewSSHClientWithCluster(usedCluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(sshClient).NotTo(gomega.BeNil())
	return &SSHClient{
		RemoteHostIP: sshClient.Host,
		SSH:          sshClient.SSH,
	}
}

func IsFileExist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func UnmarshalYamlFile(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	err = yaml.Unmarshal(data, obj)
	return err
}

func MarshalYamlToFile(file string, obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	if err = WriteFile(file, data); err != nil {
		return err
	}
	return nil
}
