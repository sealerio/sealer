package test

import (
	"github.com/alibaba/sealer/test/suites/image"
	"github.com/alibaba/sealer/test/testhelper"

	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer images", func() {
	Context("find list", func() {
		BeforeEach(func() {
			image.DoImageOps("pull", "kubernetes:v1.19.9")
		})
		AfterEach(func() {
			image.DoImageOps("rmi", "kubernetes:v1.19.9")
		})
		It("output list", func() {
			sess, err := testhelper.Start("sealer images")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
		})
	})
})
