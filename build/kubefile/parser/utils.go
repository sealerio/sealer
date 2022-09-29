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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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

			files, err := ioutil.ReadDir(source)
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
