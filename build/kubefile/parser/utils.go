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

	"github.com/sirupsen/logrus"
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

// getApplicationType:
// walk through files to copy,try to find obvious application type,
// we only support helm,kube,shell at present, and it is a strict match file suffix.
// if not found, will return "".
func getApplicationType(sources []string) (string, error) {
	isHelmType, helmErr := isHelm(sources...)
	if helmErr != nil {
		return "", helmErr
	}

	if isHelmType {
		return application.HelmApp, nil
	}

	appTypeFunc := func(fileName string) string {
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext == ".sh" {
			return application.ShellApp
		}

		if ext == ".yaml" || ext == ".yml" {
			return application.KubeApp
		}

		return application.UnknownApp
	}

	var appTypeList []string
	for _, source := range sources {
		s, err := os.Stat(source)
		if err != nil {
			return "", fmt.Errorf("failed to stat %s: %v", source, err)
		}

		// get app type by dir
		if s.IsDir() {
			err = filepath.Walk(source, func(path string, f fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if f.IsDir() {
					return nil
				}

				appTypeList = append(appTypeList, appTypeFunc(f.Name()))
				return nil
			})

			if err != nil {
				return "", fmt.Errorf("failed to walk copy dir %s: %v", source, err)
			}
			continue
		}

		// get app type by file
		appTypeList = append(appTypeList, appTypeFunc(source))
	}

	//matches the file suffix strictly
	var isShell, isKube bool
	for _, appType := range appTypeList {
		if appType == application.UnknownApp {
			logrus.Debugf("application type not detected in %s,%s,%s",
				application.KubeApp, application.HelmApp, application.ShellApp)
			return "", nil
		}

		if appType == application.ShellApp {
			isShell = true
		}

		if appType == application.KubeApp {
			isKube = true
		}
	}

	if isShell && !isKube {
		return application.ShellApp, nil
	}

	if isKube && !isShell {
		return application.KubeApp, nil
	}

	return "", nil
}

// getApplicationFiles: get application files
func getApplicationFiles(appName, appType string, sources []string) ([]string, error) {
	if appType == "" {
		return nil, nil
	}

	if appType == application.HelmApp {
		return []string{appName}, nil
	}

	var launchFiles []string

	for _, source := range sources {
		s, err := os.Stat(source)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %v", source, err)
		}

		// get app launchFile if source is a dir
		if s.IsDir() {
			err = filepath.Walk(source, func(path string, f fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if f.IsDir() {
					return nil
				}

				launchFiles = append(launchFiles, strings.TrimPrefix(path, source))
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk application dir %s: %v", source, err)
			}

			// if type is shell, only use first build context
			if appType == application.ShellApp {
				return launchFiles, nil
			}
			continue
		}
		// get app launchFile if source is a file
		launchFiles = append(launchFiles, filepath.Base(source))

		// if type is shell, only use first build context
		if appType == application.ShellApp {
			return launchFiles, nil
		}
	}

	return launchFiles, nil
}
