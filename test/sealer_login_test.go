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
	. "github.com/onsi/ginkgo"

	"github.com/sealerio/sealer/test/suites/image"
	"github.com/sealerio/sealer/test/suites/registry"
	"github.com/sealerio/sealer/test/testhelper/settings"
)

var _ = Describe("sealer login", func() {
	Context("login docker registry", func() {
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
})
