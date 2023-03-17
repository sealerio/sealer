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

	"github.com/sealerio/sealer/test/suites/build"
	"github.com/sealerio/sealer/test/suites/image"
	"github.com/sealerio/sealer/test/suites/registry"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sealer image", func() {
	Context("pull image", func() {

		It(fmt.Sprintf("pull image %s", settings.TestImageName), func() {
			image.DoImageOps("pull", settings.TestImageName)
			testhelper.CheckBeTrue(build.CheckIsImageExist(settings.TestImageName))

			tagImageNames := []string{
				"e2eimage_test:latest",
				"e2eimage_test:v0.0.1",
				"sealer-io/e2eimage_test:v0.0.2",
				"docker.io/sealerio/e2eimage_test:v0.0.3",
			}
			By("tag by image name", func() {
				for _, newOne := range tagImageNames {
					image.TagImages(settings.TestImageName, newOne)
					Expect(build.CheckIsImageExist(newOne)).Should(BeTrue())
				}

				image.DoImageOps("images", "")

				for _, imageName := range tagImageNames {
					removeImage := imageName
					image.DoImageOps("rmi", removeImage)
				}

			})

			By("remove tag image", func() {
				tagImageName := "e2e_images_test:v0.3"
				image.DoImageOps("pull", settings.TestImageName)
				image.TagImages(settings.TestImageName, tagImageName)
				testhelper.CheckBeTrue(build.CheckIsImageExist(tagImageName))
				image.DoImageOps("rmi", tagImageName)
				testhelper.CheckNotBeTrue(build.CheckIsImageExist(tagImageName))
			})

		})

		faultImageNames := []string{
			fmt.Sprintf("%s/%s:latest", settings.DefaultImageName, settings.DefaultImageRepo),
			fmt.Sprintf("%s:latest", settings.DefaultImageDomain),
			fmt.Sprintf("%s:latest", settings.DefaultImageRepo),
		}

		for _, faultImageName := range faultImageNames {
			faultImageName := faultImageName
			It(fmt.Sprintf("pull fault image %s", faultImageName), func() {
				sess, err := testhelper.Start(fmt.Sprintf("%s pull %s", settings.DefaultSealerBin, faultImageName))
				testhelper.CheckErr(err)
				testhelper.CheckNotExit0(sess, settings.DefaultWaiteTime)
				testhelper.CheckNotBeTrue(build.CheckIsImageExist(faultImageName))
			})
		}

	})

	Context("remove image", func() {
		It(fmt.Sprintf("remove image %s", settings.TestImageName), func() {
			image.DoImageOps("images", "")
			image.DoImageOps("pull", settings.TestImageName)
			testhelper.CheckBeTrue(build.CheckIsImageExist(settings.TestImageName))
			image.DoImageOps("rmi", settings.TestImageName)
			testhelper.CheckNotBeTrue(build.CheckIsImageExist(settings.TestImageName))
		})

	})

	Context("push image", func() {
		BeforeEach(func() {
			registry.Login()
			image.DoImageOps("pull", settings.TestImageName)
		})
		AfterEach(func() {
			registry.Logout()
		})
		It("push image", func() {
			pushImageName := "docker.io/sealerio/e2eimage_test:v0.0.1"
			if settings.RegistryURL != "" && settings.RegistryUsername != "" && settings.RegistryPasswd != "" {
				pushImageName = settings.RegistryURL + "/" + settings.RegistryUsername + "/" + "e2eimage_test:v0.0.1"
			}
			image.TagImages(settings.TestImageName, pushImageName)
			image.DoImageOps("push", pushImageName)
		})
	})

	Context("login registry", func() {
		AfterEach(func() {
			registry.Logout()
		})
		It("with correct name and password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryUsername,
				settings.RegistryPasswd,
				true)
		})
		It("with incorrect name and password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryPasswd,
				settings.RegistryUsername,
				false)
		})
		It("with only name", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryUsername,
				"",
				false)
		})
		It("with only password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				"",
				settings.RegistryPasswd,
				false)
		})
		It("with only registryURL", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				"",
				"",
				false)
		})
	})

	//todo add mount and umount e2e test
})
