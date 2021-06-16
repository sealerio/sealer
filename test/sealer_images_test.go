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

	"github.com/alibaba/sealer/test/suites/image"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer image", func() {
	Context("pull image", func() {
		pullImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9",
			"registry.cn-qingdao.aliyuncs.com/kubernetes:v1.19.9",
			"sealer-io/kubernetes:v1.19.9",
			"kubernetes:v1.19.9",
		}

		for _, imageName := range pullImageNames {
			imageName := imageName
			It(fmt.Sprintf("pull image %s", imageName), func() {
				image.DoImageOps(settings.SubCmdPullOfSealer, imageName)
				image.DoImageOps(settings.SubCmdRmiOfSealer, imageName)
			})
		}

		faultImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io:latest",
			"registry.cn-qingdao.aliyuncs.com:latest",
			"sealer-io:latest",
		}

		for _, faultImageName := range faultImageNames {
			faultImageName := faultImageName
			It(fmt.Sprintf("pull fault image %s", faultImageName), func() {
				sess, err := testhelper.Start(fmt.Sprintf("%s %s %s", settings.DefaultSealerBin, settings.SubCmdPullOfSealer, faultImageName))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).ShouldNot(Exit(0))
			})
		}

	})

	Context("remove image", func() {
		It(fmt.Sprintf("remove image %s", settings.TestImageName), func() {
			image.DoImageOps(settings.SubCmdListOfSealer, "")

			beforeEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("beforeEnvMd5 is %s", beforeEnvMd5))
			Expect(beforeEnvMd5).NotTo(Equal(""))
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
			afterEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
			Expect(beforeEnvMd5).To(Equal(afterEnvMd5))
		})

		It("remove tag image", func() {
			tagImageName := "e2e_image_test:v0.01"
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)

			beforeEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("beforeEnvMd5 is %s", beforeEnvMd5))
			Expect(beforeEnvMd5).NotTo(Equal(""))
			image.TagImages(settings.TestImageName, tagImageName)

			image.DoImageOps(settings.SubCmdRmiOfSealer, tagImageName)

			afterEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
			Expect(afterEnvMd5).To(Equal(beforeEnvMd5))

			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
		})
	})

	Context("push image", func() {
		BeforeEach(func() {
			registry.Login()
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
		})
		AfterEach(func() {
			registry.Logout()
			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
		})
		pushImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io/e2e_image_test:v0.01",
			"sealer-io/e2e_image_test:v0.01",
			"e2e_image_test:v0.01",
		}

		for _, pushImage := range pushImageNames {
			pushImage := pushImage
			It(fmt.Sprintf("push image %s", pushImage), func() {
				image.TagImages(settings.TestImageName, pushImage)
				image.DoImageOps(settings.SubCmdPushOfSealer, pushImage)
				image.DoImageOps(settings.SubCmdRmiOfSealer, pushImage)
			})
		}
	})
})
