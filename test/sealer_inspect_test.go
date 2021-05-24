package test

import (
	"fmt"

	"github.com/alibaba/sealer/test/suites/image"

	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer inspect", func() {
	Context("inspect image", func() {
		BeforeEach(func() {
			image.DoImageOps("pull", settings.TestImageName)
		})
		AfterEach(func() {
			image.DoImageOps("rmi", settings.TestImageName)
		})
		Context("show image metadata", func() {
			It("show image metadata", func() {
				sess, err := testhelper.Start(fmt.Sprintf("sealer inspect %s", settings.TestImageName))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
			})
		})
		Context("show image default Clusterfile", func() {
			It("show image default Clusterfile", func() {
				sess, err := testhelper.Start(fmt.Sprintf("sealer inspect -c %s", settings.TestImageName))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
			})
		})
	})
})
