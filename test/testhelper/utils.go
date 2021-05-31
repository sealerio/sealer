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
		if err != nil {
			fmt.Println(err)
		}
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
