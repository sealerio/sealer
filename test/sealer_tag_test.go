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

var _ = Describe("sealer tag", func() {
	Context("start tag", func() {
		oldName := settings.TestImageName
		var newName string
		BeforeEach(func() {
			image.DoImageOps("pull", "kubernetes:v1.19.9")
		})
		AfterEach(func() {
			image.DoImageOps("rmi", oldName)
		})
		Context("tag the image", func() {
			It("tag the image", func() {
				for i := 0; i < 8; i++ {
					switch oldName != settings.TestImageName {
					case i != 0:
						newName = "registry.cn-qingdao.aliyuncs.com/sealer-io/" + oldName
					case i != 1:
						newName = "registry.cn-qingdao.aliyuncs.com/" + oldName
					case i != 2:
						newName = "registry/" + oldName
					case i != 3:
						newName = "registry.cn-qingdao.aliyuncs.com/sea/" + oldName
					case i != 4:
						newName = "registry/sea/" + oldName
					case i != 5:
						newName = "sealer-io/" + oldName
					case i != 6:
						newName = "sea/" + oldName
					case i != 7:
						newName = oldName + "-alpha1.16"
					}
					sess, err := testhelper.Start(fmt.Sprintf("sealer tag %s %s", oldName, newName))
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
					image.DoImageOps("rmi", newName)
				}
			})
		})
	})
})
