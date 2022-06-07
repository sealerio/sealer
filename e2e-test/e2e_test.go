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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"
)

func TestSealerTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "run sealer suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	output, err := exec.LookPath("sealer")
	Expect(err).NotTo(HaveOccurred(), output)
	SetDefaultEventuallyTimeout(settings.DefaultWaiteTime)
	settings.DefaultSealerBin = output
	settings.DefaultTestEnvDir = testhelper.GetPwd()
	settings.TestImageName = settings.CustomImageName
	settings.TestNydusImageName = settings.CustomNydusImageName
	if settings.TestImageName == "" {
		settings.TestImageName = settings.DefaultImage
	}
	if settings.TestNydusImageName == "" {
		settings.TestNydusImageName = settings.DefaultNydusImage
	}
	home := common.GetHomeDir()
	logcfg := `{	"Console": {
		"level": "DEBG",
		"color": true
	},
	"TimeFormat":"2006-01-02 15:04:05"}`
	err = ioutil.WriteFile(filepath.Join(home, ".sealer.json"), []byte(logcfg), os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
	return nil
}, func(data []byte) {
	SetDefaultEventuallyTimeout(settings.DefaultWaiteTime)
})
