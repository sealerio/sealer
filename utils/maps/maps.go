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

package maps

// ConvertToSlice Use the equal sign to link key and value looks like key1=value1,key2=value2.
func ConvertToSlice(m map[string]string) []string {
	result := []string{}
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

// Merge :get all elements, only insert key which is not in dst form src.
func Merge(dst, src map[string]string) map[string]string {
	if len(dst) == 0 {
		return Copy(src)
	}
	for srcEnvKey, srcEnvValue := range src {
		if _, ok := dst[srcEnvKey]; ok {
			continue
		}
		dst[srcEnvKey] = srcEnvValue
	}
	return dst
}

func Copy(origin map[string]string) map[string]string {
	if origin == nil {
		return nil
	}
	ret := make(map[string]string, len(origin))
	for k, v := range origin {
		ret[k] = v
	}

	return ret
}
