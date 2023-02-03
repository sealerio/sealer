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

package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/gomega"

	"github.com/sealerio/sealer/test/testhelper/settings"
	"github.com/sealerio/sealer/utils/exec"
)

func GetFixtures() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures")
}

func GetLiteBuildDir() string {
	return "lite_build"
}

func GetCloudBuildDir() string {
	return "cloud_build"
}

func GetContainerBuildDir() string {
	return "container_build"
}

// GetBuildImageName return specific image name for sealer build test
func GetBuildImageName() string {
	return fmt.Sprintf("%s-%s", settings.TestImageName, "forBuildTest")
}

type ArgsOfBuild struct {
	KubeFile, ImageName, Context string
}

func (a *ArgsOfBuild) SetKubeFile(kubeFile string) *ArgsOfBuild {
	a.KubeFile = kubeFile
	return a
}

func (a *ArgsOfBuild) SetImageName(imageName string) *ArgsOfBuild {
	a.ImageName = imageName
	return a
}

func (a *ArgsOfBuild) SetContext(context string) *ArgsOfBuild {
	a.Context = context
	return a
}

func (a *ArgsOfBuild) Build() string {
	if settings.DefaultSealerBin == "" || a.KubeFile == "" || a.ImageName == "" {
		return ""
	}

	if a.Context == "" {
		a.Context = "."
	}

	return fmt.Sprintf("%s build -f %s -t %s %s -d", settings.DefaultSealerBin, a.KubeFile, a.ImageName, a.Context)
}

func NewArgsOfBuild() *ArgsOfBuild {
	return &ArgsOfBuild{}
}

func CheckIsImageExist(imageName string) bool {
	cmd := fmt.Sprintf("%s inspect %s", settings.DefaultSealerBin, imageName)
	_, err := exec.RunSimpleCmd(cmd)
	return err == nil
}

func UpdateKubeFromImage(imageName string, KubefilePath string) {
	Kube, err := os.ReadFile(filepath.Clean(KubefilePath))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	Kube = append([]byte(fmt.Sprintf("FROM %s", imageName)), Kube[bytes.IndexByte(Kube, '\n'):]...) // #nosec
	err = os.WriteFile(KubefilePath, Kube, os.ModePerm)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}
