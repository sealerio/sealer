package testhelper

import (
	"github.com/onsi/gomega"
	"os"
)

func GetPwd() string {
	pwd, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pwd
}
