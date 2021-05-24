package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func Login() {
	// check if docker json already exist
	config := filepath.Join(settings.DefaultRegistryAuthDir, "config.json")
	if utils.IsFileExist(config) {
		return
	}
	sess, err := testhelper.Start(fmt.Sprintf("sealer login %s -u %s -p %s", settings.RegistryURL,
		settings.RegistryUsername,
		settings.RegistryPasswd))

	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(sess).Should(gbytes.Say(fmt.Sprintf("login %s success", settings.RegistryURL)))
	gomega.Eventually(sess, settings.MaxWaiteTime).Should(gexec.Exit(0))
}

func Logout() {
	err := CleanLoginFile()
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func CleanLoginFile() error {
	return os.RemoveAll(settings.DefaultRegistryAuthDir)
}
