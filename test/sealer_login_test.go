package test

import (
	"fmt"

	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer login", func() {
	Context("login docker registry", func() {
		AfterEach(func() {
			registry.Logout()
		})
		It("with correct name and password", func() {
			sess, err := testhelper.Start(fmt.Sprintf("%s login %s -u %s -p %s", settings.DefaultSealerBin, settings.RegistryURL,
				settings.RegistryUsername, settings.RegistryPasswd))
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say(fmt.Sprintf("login %s success", settings.RegistryURL)))
			Eventually(sess).Should(Exit(0))
		})
	})
})
