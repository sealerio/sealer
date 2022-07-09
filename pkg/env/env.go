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
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

const templateSuffix = ".tmpl"

type Interface interface {
	PreProcessor
	// WrapperShell :If host already set env like DATADISK=/data
	// This function add env to the shell, like:
	// Input shell: cat /etc/hosts
	// Output shell: DATADISK=/data cat /etc/hosts
	// So that you can get env values in you shell script
	WrapperShell(host net.IP, shell string) string
	// RenderAll :render env to all the files in dir
	RenderAll(host net.IP, dir string) error
}

type processor struct {
	*v2.Cluster
}

func NewEnvProcessor(cluster *v2.Cluster) Interface {
	return &processor{cluster}
}

func (p *processor) WrapperShell(host net.IP, shell string) string {
	var env string
	for k, v := range p.getHostEnv(host) {
		switch value := v.(type) {
		case []string:
			env = fmt.Sprintf("%s%s=(%s) ", env, k, strings.Join(value, " "))
		case string:
			env = fmt.Sprintf("%s%s=%s ", env, k, value)
		}
	}
	if env == "" {
		return shell
	}
	return fmt.Sprintf("%s && %s", env, shell)
}

func (p *processor) RenderAll(host net.IP, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, errIn error) error {
		if errIn != nil {
			return errIn
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), templateSuffix) {
			return nil
		}
		writer, err := os.OpenFile(strings.TrimSuffix(path, templateSuffix), os.O_CREATE|os.O_RDWR, os.ModePerm)
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
		if err := t.Execute(writer, p.getHostEnv(host)); err != nil {
			return fmt.Errorf("failed to render env template(%s): %v", path, err)
		}
		return nil
	})
}

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

func mergeList(hostEnv, globalEnv map[string]interface{}) map[string]interface{} {
	if len(hostEnv) == 0 {
		return globalEnv
	}
	for globalEnvKey, globalEnvValue := range globalEnv {
		if _, ok := hostEnv[globalEnvKey]; ok {
			continue
		}
		hostEnv[globalEnvKey] = globalEnvValue
	}
	return hostEnv
}

// Merge the host ENV and global env, the host env will overwrite cluster.Spec.Env
func (p *processor) getHostEnv(hostIP net.IP) (env map[string]interface{}) {
	hostEnv, globalEnv := map[string]interface{}{}, ConvertEnv(p.Spec.Env)

	for _, host := range p.Spec.Hosts {
		for _, ip := range host.IPS {
			if ip.Equal(hostIP) {
				hostEnv = ConvertEnv(host.Env)
			}
		}
	}
	return mergeList(hostEnv, globalEnv)
}

// ConvertEnv []string to map[string]interface{}, example [IP=127.0.0.1,IP=192.160.0.2,Key=value] will convert to {IP:[127.0.0.1,192.168.0.2],key:value}
func ConvertEnv(envList []string) (env map[string]interface{}) {
	temp := make(map[string][]string)
	env = make(map[string]interface{})

	for _, e := range envList {
		var kv []string
		if kv = strings.SplitN(e, "=", 2); len(kv) != 2 {
			continue
		}
		temp[kv[0]] = append(temp[kv[0]], strings.Split(kv[1], ";")...)
	}

	for k, v := range temp {
		if len(v) > 1 {
			env[k] = v
			continue
		}
		if len(v) == 1 {
			env[k] = v[0]
		}
	}

	return
}
