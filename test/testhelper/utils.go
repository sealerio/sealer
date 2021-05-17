package testhelper

import (
	"os"

	"github.com/onsi/gomega"
)

func GetPwd() string {
	pwd, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pwd
}
