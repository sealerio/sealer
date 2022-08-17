// Copyright © 2022 Alibaba Group Holding Ltd.
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

package parser

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sealerio/sealer/pkg/define/application"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/sealerio/sealer/pkg/define/application/version"

	v1 "github.com/sealerio/sealer/pkg/define/application/v1"

	"github.com/stretchr/testify/assert"
)

const (
//nginxDemoURL = "https://raw.githubusercontent.com/kubernetes/website/main/content/en/examples/application/deployment.yaml"

)

var testParser *KubefileParser

var testAppRootPath = "/test/apps"

func TestParserKubeApp(t *testing.T) {
	nginxDemoPath := "./test/kube-nginx-deployment/deployment.yaml"
	imageEngine := testImageEngine{}
	opts := options.BuildOptions{
		PullPolicy: "missing",
	}

	buildCxt, err := setupTempContext()
	assert.Equal(t, nil, err)
	defer func() {
		_ = os.RemoveAll(buildCxt)
	}()

	opts.ContextDir = buildCxt
	testParser = NewParser(testAppRootPath, opts, imageEngine)

	var (
		app1Name = "nginx"
		app1Path = testParser.appRootPathFunc(app1Name)
		text     = fmt.Sprintf(`
FROM busybox as base
APP %s local://%s
`, app1Name, nginxDemoPath)
	)

	reader := bytes.NewReader([]byte(text))
	result, err := testParser.ParseKubefile(reader)
	if err != nil {
		t.Fatalf("failed to parse kubefile: %s", err)
	}
	defer func() {
		_ = result.CleanLegacyContext()
	}()

	var expectedText = fmt.Sprintf(`
FROM busybox as base
copy %s %s
`,
		strings.Join(result.legacyContext.apps2Files[app1Name], " "),
		app1Path,
	)

	result.Dockerfile = strings.TrimSpace(result.Dockerfile)
	expectedResult := &KubefileResult{
		Dockerfile: strings.TrimSpace(expectedText),
		LaunchList: []string{},
		Applications: map[string]version.VersionedApplication{
			app1Name: v1.NewV1Application(
				app1Name,
				application.KubeApp,
			),
		},
	}

	assert.Equal(t, expectedResult.Dockerfile, result.Dockerfile)
	assert.Equal(t, len(expectedResult.Applications), len(result.Applications))
	assert.Equal(t, expectedResult.Applications[app1Name], result.Applications[app1Name])
	assert.Equal(t, expectedResult.LaunchList, result.LaunchList)
}

func TestParserHelmApp(t *testing.T) {
	githubAppPath := "./test/brigade-github-app"
	imageEngine := testImageEngine{}
	opts := options.BuildOptions{
		PullPolicy: "missing",
	}

	buildCxt, err := setupTempContext()
	assert.Equal(t, nil, err)
	defer func() {
		_ = os.RemoveAll(buildCxt)
	}()

	opts.ContextDir = buildCxt
	testParser = NewParser(testAppRootPath, opts, imageEngine)

	var (
		app1Name = "github-app"
		app1Path = testParser.appRootPathFunc(app1Name)
		text     = fmt.Sprintf(`
FROM busybox as base
APP %s local://%s
LAUNCH %s
`, app1Name, githubAppPath, app1Name)
	)

	reader := bytes.NewReader([]byte(text))
	result, err := testParser.ParseKubefile(reader)
	if err != nil {
		t.Fatalf("failed to parse kubefile: %s", err)
	}
	defer func() {
		_ = result.CleanLegacyContext()
	}()

	var expectedText = fmt.Sprintf(`
FROM busybox as base
copy %s %s
`,
		strings.Join(result.legacyContext.apps2Files[app1Name], " "),
		app1Path,
	)

	result.Dockerfile = strings.TrimSpace(result.Dockerfile)
	expectedResult := &KubefileResult{
		Dockerfile: strings.TrimSpace(expectedText),
		LaunchList: []string{
			fmt.Sprintf("helm install %s %s", app1Name, app1Path),
		},
		Applications: map[string]version.VersionedApplication{
			app1Name: v1.NewV1Application(
				app1Name,
				application.HelmApp,
			),
		},
	}

	assert.Equal(t, expectedResult.Dockerfile, result.Dockerfile)
	assert.Equal(t, len(expectedResult.Applications), len(result.Applications))
	assert.Equal(t, expectedResult.Applications[app1Name], result.Applications[app1Name])
	assert.Equal(t, expectedResult.LaunchList, result.LaunchList)
}

func TestParserCMDS(t *testing.T) {
	imageEngine := testImageEngine{}
	opts := options.BuildOptions{
		PullPolicy: "missing",
	}

	buildCxt, err := setupTempContext()
	assert.Equal(t, nil, err)
	defer func() {
		_ = os.RemoveAll(buildCxt)
	}()

	opts.ContextDir = buildCxt
	testParser = NewParser(testAppRootPath, opts, imageEngine)

	var (
		text = fmt.Sprintf(`
FROM busybox as base
CMDS ["%s", "%s"]
`, "kubectl apply -f abc.yaml", "kubectl apply -f bcd.yaml")
	)

	reader := bytes.NewReader([]byte(text))
	result, err := testParser.ParseKubefile(reader)
	if err != nil {
		t.Fatalf("failed to parse kubefile: %s", err)
	}
	defer func() {
		_ = result.CleanLegacyContext()
	}()

	var expectedText = `
FROM busybox as base
`

	result.Dockerfile = strings.TrimSpace(result.Dockerfile)
	expectedResult := &KubefileResult{
		Dockerfile: strings.TrimSpace(expectedText),
		LaunchList: []string{
			"kubectl apply -f abc.yaml",
			"kubectl apply -f bcd.yaml",
		},
		Applications: map[string]version.VersionedApplication{},
	}

	assert.Equal(t, expectedResult.Dockerfile, result.Dockerfile)
	assert.Equal(t, len(expectedResult.Applications), len(result.Applications))
	assert.Equal(t, expectedResult.LaunchList, result.LaunchList)
}

func setupTempContext() (string, error) {
	tmpDir, err := os.MkdirTemp("/tmp/", "sealer-test")
	if err != nil {
		return "", err
	}

	return tmpDir, nil
}
