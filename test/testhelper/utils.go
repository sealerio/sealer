package testhelper

import (
	"fmt"
	"github.com/onsi/gomega"
	"io/ioutil"
	"k8s.io/client-go/util/homedir"
	"os"
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

func GetUsedClusterFilePath(clusterName string) string {
	if home := homedir.HomeDir(); home != "" {
		return fmt.Sprintf("%s/.sealer/%s/Clusterfile", home, clusterName)
	}
	return ""
}
