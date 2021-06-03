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
			err := registry.CleanLoginFile()
			Expect(err).NotTo(HaveOccurred())
		})
		It("with correct name and password", func() {
			sess, err := testhelper.Start(fmt.Sprintf("sealer login %s -u %s -p %s", settings.RegistryURL,
				settings.RegistryUsername, settings.RegistryPasswd))
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say(fmt.Sprintf("login %s success", settings.RegistryURL)))
			Eventually(sess).Should(Exit(0))
		})
	})
})
