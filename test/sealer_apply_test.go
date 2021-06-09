// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

/*import (
	"fmt"

	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {
		BeforeEach(func() {
			registry.Login()
		})

		Context("with roofs images", func() {
			clusterFile := apply.GetClusterFilePathOfRootfs()
			AfterEach(func() {
				cluster := apply.GetClusterFileData(clusterFile)
				apply.DeleteCluster(cluster.ClusterName)
			})

			It("apply cluster", func() {
				sess, err := testhelper.Start(fmt.Sprintf("sealer apply -f %s", clusterFile))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
			})
		})

	})

})*/
