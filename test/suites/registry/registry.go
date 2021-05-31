package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func Login() {
	// check if docker json already exist
	config := DefaultRegistryAuthConfigDir()
	if utils.IsFileExist(config) {
		return
	}
	sess, err := testhelper.Start(fmt.Sprintf("%s login %s -u %s -p %s", settings.DefaultSealerBin, settings.RegistryURL,
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
	return os.RemoveAll(DefaultRegistryAuthConfigDir())
}
func DefaultRegistryAuthConfigDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		return settings.DefaultRegistryAuthFile
	}

	return filepath.Join(dir, ".docker/config.json")
}
