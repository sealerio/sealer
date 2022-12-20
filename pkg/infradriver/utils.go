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

package infradriver

import (
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
)

const (
	DelSymbol   = "-"
	EqualSymbol = "="
	ColonSymbol = ":"
)

func newTaintStruct(key, value, effect string) k8sv1.Taint {
	return k8sv1.Taint{Key: key, Value: value, Effect: k8sv1.TaintEffect(effect)}
}

// FormatData process data in the specified format
//eg: key1=value1:NoSchedule;key1:NoSchedule;key1=value1:NoSchedule
func formatData(data string) ([]k8sv1.Taint, error) {
	var (
		key, value, effect string
		taints             []k8sv1.Taint
		TaintEffectValues  = []k8sv1.TaintEffect{k8sv1.TaintEffectNoSchedule, k8sv1.TaintEffectNoExecute, k8sv1.TaintEffectPreferNoSchedule}
	)

	data = strings.TrimSpace(data)
	switch {
	case strings.Contains(data, EqualSymbol) && !strings.Contains(data, EqualSymbol+ColonSymbol):
		temps := strings.Split(data, EqualSymbol)
		if len(temps) != 2 {
			return nil, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key = temps[0]
		taintArgs := strings.Split(temps[1], ColonSymbol)
		if len(taintArgs) != 2 {
			return nil, fmt.Errorf("error: invalid taint data: %s", data)
		}
		value, effect = taintArgs[0], taintArgs[1]

	case !strings.Contains(data, EqualSymbol) && strings.Contains(data, ColonSymbol):
		temps := strings.Split(data, ColonSymbol)
		if len(temps) != 2 {
			return nil, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key, value, effect = temps[0], "", temps[1]

	case strings.Contains(data, EqualSymbol+ColonSymbol):
		temps := strings.Split(data, EqualSymbol+ColonSymbol)
		if len(temps) != 2 {
			return nil, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key, value, effect = temps[0], "", temps[1]

	// key1-
	case strings.Contains(data, DelSymbol) && !strings.Contains(data, EqualSymbol+ColonSymbol):
		key, value, effect = data, "", ""
	}

	if effect != "" {
		taintEffect := strings.TrimSuffix(effect, DelSymbol)
		if notInEffect(k8sv1.TaintEffect(taintEffect), TaintEffectValues) {
			return nil, fmt.Errorf("taint effect %s need in %v", data, TaintEffectValues)
		}
	}

	taint := newTaintStruct(key, value, effect)
	taints = append(taints, taint)
	return taints, nil
}

func notInEffect(effect k8sv1.TaintEffect, effects []k8sv1.TaintEffect) bool {
	for _, e := range effects {
		if e == effect {
			return false
		}
	}
	return true
}
