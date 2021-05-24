package testhelper

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/onsi/gomega"
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
