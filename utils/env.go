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

package utils

import (
	"fmt"
	"os"
	"strings"
)

func SetRootfsBinToSystemEnv(rootfs string) error {
	bin := fmt.Sprintf(":%s/bin", rootfs)
	return os.Setenv("PATH", os.Getenv("PATH")+bin)
}

func ConvertMapToEnvList(m map[string]string) []string {
	result := []string{}
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

func ConvertEnvListToMap(env []string) map[string]string {
	envs := map[string]string{}
	for _, e := range env {
		i := strings.Index(e, "=")

		if i < 0 {
			envs[e] = ""
		} else {
			k := e[:i]
			v := e[i+1:]
			envs[k] = v
		}
	}
	return envs
}
