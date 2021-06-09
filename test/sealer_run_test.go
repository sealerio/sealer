package test

import (
	"fmt"

	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer run", func() {
	Context("start run", func() {
		Context("regular scenario", func() {
			Context("run on ali cloud", func() {
				AfterEach(func() {
					apply.DeleteClusterByFile(testhelper.GetRootClusterFilePath(settings.ClusterNameForRun))
				})

				It("exec sealer run", func() {
					master := "1"
					node := "1"
					cmd := fmt.Sprintf("%s run %s -m %s -n %s", settings.DefaultSealerBin, settings.ImageNameForRun, master, node)
					sess, err := testhelper.Start(cmd)
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
					apply.CheckNodeNumLocally(2)
				})

			})

		})

	})

})
