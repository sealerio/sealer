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

/*
add:
key1=value1:NoSchedule;
key1:NoSchedule;
key1=:NoSchedule

delete:
key-;
key1:NoSchedule-;
*/
// formatData process data in the specified format
func formatData(data string) (k8sv1.Taint, error) {
	var (
		key, value, effect string
		TaintEffectValues  = []k8sv1.TaintEffect{k8sv1.TaintEffectNoSchedule, k8sv1.TaintEffectNoExecute, k8sv1.TaintEffectPreferNoSchedule}
	)

	data = strings.TrimSpace(data)
	switch {
	// key1=value1:NoSchedule
	case strings.Contains(data, EqualSymbol) && !strings.Contains(data, EqualSymbol+ColonSymbol) && !strings.Contains(data, DelSymbol):
		temps := strings.Split(data, EqualSymbol)
		if len(temps) != 2 {
			return k8sv1.Taint{}, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key = temps[0]
		taintArgs := strings.Split(temps[1], ColonSymbol)
		if len(taintArgs) != 2 {
			return k8sv1.Taint{}, fmt.Errorf("error: invalid taint data: %s", data)
		}
		value, effect = taintArgs[0], taintArgs[1]
		effect = strings.TrimSuffix(effect, DelSymbol)

		//key1:NoSchedule
	case !strings.Contains(data, EqualSymbol) && strings.Contains(data, ColonSymbol) && !strings.Contains(data, DelSymbol):
		temps := strings.Split(data, ColonSymbol)
		if len(temps) != 2 {
			return k8sv1.Taint{}, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key, value, effect = temps[0], "", temps[1]

		//key1=:NoSchedule
	case strings.Contains(data, EqualSymbol+ColonSymbol) && !strings.Contains(data, DelSymbol):
		temps := strings.Split(data, EqualSymbol+ColonSymbol)
		if len(temps) != 2 {
			return k8sv1.Taint{}, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key, value, effect = temps[0], "", temps[1]

		// key1-
	case strings.Contains(data, DelSymbol) && !strings.Contains(data, EqualSymbol) && !strings.Contains(data, ColonSymbol):
		key, value, effect = data, "", ""

		// key1:NoSchedule-
	case strings.Contains(data, DelSymbol) && !strings.Contains(data, EqualSymbol) && strings.Contains(data, ColonSymbol):
		temps := strings.Split(data, ColonSymbol)
		if len(temps) != 2 {
			return k8sv1.Taint{}, fmt.Errorf("faild to split taint argument: %s", data)
		}
		key, value, effect = temps[0], "", temps[1]
	}

	//determine whether the Effect is legal
	if effect != "" {
		taintEffect := strings.TrimSuffix(effect, DelSymbol)
		if notInEffect(k8sv1.TaintEffect(taintEffect), TaintEffectValues) {
			return k8sv1.Taint{}, fmt.Errorf("taint effect %s need in %v", data, TaintEffectValues)
		}
	}

	taint := k8sv1.Taint{
		Key:    key,
		Value:  value,
		Effect: k8sv1.TaintEffect(effect),
	}
	return taint, nil
}

func notInEffect(effect k8sv1.TaintEffect, effects []k8sv1.TaintEffect) bool {
	for _, e := range effects {
		if e == effect {
			return false
		}
	}
	return true
}

// DeleteTaintsByKey removes all the taints that have the same key to given taintKey
func DeleteTaintsByKey(taints []k8sv1.Taint, taintKey string) ([]k8sv1.Taint, bool) {
	newTaints := []k8sv1.Taint{}
	for i := range taints {
		if taintKey == taints[i].Key {
			continue
		}
		newTaints = append(newTaints, taints[i])
	}
	return newTaints, len(taints) != len(newTaints)
}

// DeleteTaint removes all the taints that have the same key and effect to given taintToDelete.
func DeleteTaint(taints []k8sv1.Taint, taintToDelete *k8sv1.Taint) ([]k8sv1.Taint, bool) {
	newTaints := []k8sv1.Taint{}
	for i := range taints {
		if taintToDelete.MatchTaint(&taints[i]) {
			continue
		}
		newTaints = append(newTaints, taints[i])
	}
	return newTaints, len(taints) != len(newTaints)
}
