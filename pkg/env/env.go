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

package env

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

const templateSuffix = ".tmpl"

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) string {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// RenderTemplate :using renderData got from clusterfile to render all the files in dir with ".tmpl" as suffix.
// The scope of renderData comes from cluster.spec.env
func RenderTemplate(dir string, renderData map[string]string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, errIn error) error {
		if errIn != nil {
			return errIn
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), templateSuffix) {
			return nil
		}
		writer, err := os.Create(strings.TrimSuffix(path, templateSuffix))
		if err != nil {
			return fmt.Errorf("failed to open file [%s] when render env: %v", path, err)
		}
		defer func() {
			_ = writer.Close()
		}()
		t, err := template.New(info.Name()).Funcs(template.FuncMap{
			"b64enc": base64encode,
			"b64dec": base64decode,
		}).ParseFiles(path)
		if err != nil {
			return fmt.Errorf("failed to create template(%s): %v", path, err)
		}
		if err := t.Execute(writer, renderData); err != nil {
			return fmt.Errorf("failed to render env template(%s): %v", path, err)
		}
		return nil
	})
}

// WrapperShell :If target host already set env like DATADISK=/data in the clusterfile,
// This function will WrapperShell cmd like:
// Input shell: cat /etc/hosts
// Output shell: DATADISK=/data cat /etc/hosts
// it is convenient for user to get env in scripts
// The scope of env comes from cluster.spec.env and host.env
func WrapperShell(shell string, wrapperData map[string]string) string {
	env := getEnvFromData(wrapperData)

	if len(env) == 0 {
		return shell
	}
	return fmt.Sprintf("%s %s", strings.Join(env, " "), shell)
}

func getEnvFromData(wrapperData map[string]string) []string {
	var env []string
	for k, v := range wrapperData {
		env = append(env, fmt.Sprintf("export %s=\"%s\";", k, v))
	}
	sort.Strings(env)
	return env
}
