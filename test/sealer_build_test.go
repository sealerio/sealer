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

import (
	"fmt"
	"os"

	"github.com/alibaba/sealer/test/suites/build"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer build", func() {
	Context("start build", func() {

		BeforeEach(func() {
			registry.Login()
		})
		AfterEach(func() {
			registry.Logout()
		})

		Context("cloud build", func() {
			Context("build with only copy cmd", func() {
				imageName := settings.GetTestImageName()
				context := build.GetOnlyCopyDir()
				BeforeEach(func() {
					err := os.Chdir(context)
					Expect(err).NotTo(HaveOccurred())
				})
				AfterEach(func() {
					// run delete test image
				})
				It("only copy", func() {
					sess, err := testhelper.Start(fmt.Sprintf("sealer build -t %s .", imageName))
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
				})
			})

			Context("build with all cmd", func() {
				imageName := settings.GetTestImageName()
				context := build.GetBuildTestDir()
				BeforeEach(func() {
					err := os.Chdir(context)
					Expect(err).NotTo(HaveOccurred())
				})
				AfterEach(func() {
					// run delete test image
				})
				It("all build cmd", func() {
					sess, err := testhelper.Start(fmt.Sprintf("sealer build -t %s .", imageName))
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
				})
			})

		})

		Context("local build", func() {
		})

	})
})
