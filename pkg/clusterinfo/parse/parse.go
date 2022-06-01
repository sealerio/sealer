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

package parse

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/logger"
)

var DefaultCheckersMap = make(map[string]Checker)

func init() {
	for _, checker := range DefaultCheckers {
		DefaultCheckersMap[checker.Name()] = checker
	}
}

func GetScriptNameFromFile(scriptFilePath string) string {
	checkerFileName := strings.ToLower(filepath.Base(scriptFilePath))
	checkerName := strings.TrimSuffix(checkerFileName, filepath.Ext(checkerFileName))
	return checkerName
}

func DumpScripts(scriptDir string) error {
	_, err := os.Stat(scriptDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(scriptDir, os.ModePerm)
		if err != nil {
			logger.Error("create default script dir failed, please create it by your self mkdir -p %s\n", scriptDir)
			return err
		}
	}

	for _, checker := range DefaultCheckers {
		scriptData, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(checker.Script(), " ", "\n"))
		if err != nil {
			logger.Error("decode script error: %s", err.Error())
			return err
		}
		script := string(scriptData)
		fileName := GetScriptPath(scriptDir, checker.Name())
		if err := ioutil.WriteFile(fileName, []byte(script), 0755); err != nil {
			logger.Error("write to file %s error: %s", fileName, err)
			return err
		}
	}

	return nil
}

func GetScriptPath(scriptDir, checkerName string) string {
	return filepath.Join(scriptDir, fmt.Sprintf("%s.sh", strings.ToLower(checkerName)))
}

func GetTmpScriptPath(checkerName string) string {
	return fmt.Sprintf("/tmp/%s.sh", strings.ToLower(checkerName))
}
