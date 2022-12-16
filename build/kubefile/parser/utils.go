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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/pkg/define/application"
	osi "github.com/sealerio/sealer/utils/os"
)

const (
	schemeLocal = "local://"
	schemeHTTP  = "http://"
	schemeHTTPS = "https://"
)

func mergeLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func makeItDir(str string) string {
	if !strings.HasSuffix(str, "/") {
		return str + "/"
	}
	return str
}

func isLocal(str string) bool {
	return strings.HasPrefix(str, schemeLocal)
}

func trimLocal(str string) string {
	return strings.TrimPrefix(str, schemeLocal)
}

func isRemote(str string) bool {
	return strings.HasPrefix(str, schemeHTTP) || strings.HasPrefix(str, schemeHTTPS)
}

func isHelm(sources ...string) (bool, error) {
	isChartsArtifactEnough := func(path string) bool {
		return osi.IsFileExist(filepath.Join(path, "Chart.yaml")) &&
			osi.IsFileExist(filepath.Join(path, "values.yaml")) &&
			osi.IsFileExist(filepath.Join(path, "templates"))
	}

	isHelmInDir := func(dirPath string) (bool, error) {
		isH := false
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			isH = isChartsArtifactEnough(path)
			return filepath.SkipDir
		})

		if err == filepath.SkipDir {
			err = nil
		}

		return isH, err
	}

	chartInTargetsRoot := 0
	oneOfChartsArtifact := func(str string) {
		switch {
		case strings.HasSuffix(str, "Chart.yaml"):
			chartInTargetsRoot |= 1
		case strings.HasSuffix(str, "values.yaml"):
			chartInTargetsRoot |= 2
		case strings.HasSuffix(str, "templates"):
			chartInTargetsRoot |= 4
		}
	}

	for _, source := range sources {
		s, err := os.Stat(source)
		if err != nil {
			return false, fmt.Errorf("failed to stat %s: %v", source, err)
		}

		if s.IsDir() {
			if isH, err := isHelmInDir(source); err != nil {
				return false, fmt.Errorf("error in calling isHelmInDir: %v", err)
			} else if isH {
				return true, nil
			}

			files, err := os.ReadDir(source)
			if err != nil {
				return false, fmt.Errorf("failed to read dir (%s) in isHelm: %s", source, err)
			}
			for _, f := range files {
				if !f.IsDir() {
					oneOfChartsArtifact(f.Name())
				}
			}
		} else {
			oneOfChartsArtifact(source)
		}
	}

	return chartInTargetsRoot == 7, nil
}

// isYaml sources slice only has one element
func isYaml(sources ...string) (bool, error) {
	isYamlType := func(fileName string) bool {
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext == ".yaml" || ext == ".yml" {
			return true
		}
		return false
	}

	for _, source := range sources {
		s, err := os.Stat(source)
		if err != nil {
			return false, fmt.Errorf("failed to stat %s: %v", source, err)
		}

		if s.IsDir() {
			isAllYamlFiles := true
			err = filepath.Walk(source, func(path string, f fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if f.IsDir() {
					return nil
				}
				// make sure all files under source dir is yaml type.
				if !isYamlType(f.Name()) {
					isAllYamlFiles = false
					return filepath.SkipDir
				}
				return nil
			})
			if err != nil {
				return false, fmt.Errorf("failed to walk yaml dir %s: %v", source, err)
			}

			if isAllYamlFiles {
				return true, nil
			}
			return false, nil
		}
		if isYamlType(source) {
			return true, nil
		}
	}

	return false, nil
}

// isShell sources slice only has one element
func isShell(sources ...string) (bool, []string, error) {
	var launchFiles []string
	isShellType := func(fileName string) bool {
		ext := strings.ToLower(filepath.Ext(fileName))
		return ext == ".sh"
	}

	for _, source := range sources {
		s, err := os.Stat(source)
		if err != nil {
			return false, nil, fmt.Errorf("failed to stat %s: %v", source, err)
		}
		if s.IsDir() {
			err = filepath.Walk(source, func(path string, f fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if f.IsDir() {
					return nil
				}
				// todo optimize: use more accurate methods to determine file types.
				if !isShellType(f.Name()) {
					return filepath.SkipDir
				}

				launchFiles = append(launchFiles, path)
				return nil
			})

			if err != nil {
				return false, nil, fmt.Errorf("failed to walk shell dir %s: %v", source, err)
			}

			if len(launchFiles) > 0 {
				return true, launchFiles, nil
			}
			return false, nil, nil
		}
		if isShellType(source) {
			return true, []string{source}, nil
		}
	}

	return false, nil, nil
}

func getApplicationType(sources []string) (string, []string, error) {
	isy, yamlErr := isYaml(sources...)
	if isy {
		return application.KubeApp, sources, nil
	}

	iss, files, shellErr := isShell(sources...)
	if iss {
		return application.ShellApp, files, nil
	}

	ish, helmErr := isHelm(sources...)
	if ish {
		return application.HelmApp, sources, nil
	}

	if yamlErr != nil {
		return "", nil, yamlErr
	}
	if shellErr != nil {
		return "", nil, shellErr
	}
	if helmErr != nil {
		return "", nil, helmErr
	}

	return "", nil, fmt.Errorf("unsupported application type in %s,%s,%s", application.KubeApp, application.HelmApp, application.ShellApp)
}
