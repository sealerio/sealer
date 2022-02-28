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

package plugin

import (
	"fmt"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"

	v1 "k8s.io/api/core/v1"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/client/k8s"
)

const (
	DelSymbol   = "-"
	EqualSymbol = "="
	ColonSymbol = ":"
)

var TaintEffectValues = []v1.TaintEffect{v1.TaintEffectNoSchedule, v1.TaintEffectNoExecute, v1.TaintEffectPreferNoSchedule}

type Taint struct {
	DelTaintList []v1.Taint
	AddTaintList []v1.Taint
}

func NewTaintPlugin() Interface {
	return &Taint{}
}

func init() {
	Register(TaintPlugin, NewTaintPlugin())
}

func newTaintStruct(key, value, effect string) v1.Taint {
	return v1.Taint{Key: key, Value: value, Effect: v1.TaintEffect(effect)}
}

//Run taint_plugin file:
//apiVersion: sealer.aliyun.com/v1alpha1
//kind: Plugin
//metadata:
//  name: taint
//spec:
//  type: Taint
//  action: PreGuest
//  'on': master ##"'on': 192.168.56.1,192.168.56.2,192.168.56.3" or "'on': 192.168.56.1-192.168.56.3
//  data: key1=value1:NoSchedule ## add taint
//  #data: key1=value1:NoSchedule- ## del taint

func (l *Taint) Run(context Context, phase Phase) error {
	if phase != PhasePreGuest || context.Plugin.Spec.Type != TaintPlugin {
		logger.Debug("label nodes is PostInstall!")
		return nil
	}
	allHostIP := append(context.Cluster.GetMasterIPList(), context.Cluster.GetNodeIPList()...)
	if on := context.Plugin.Spec.On; on != "" {
		if strings.Contains(on, EqualSymbol) {
			if phase != PhasePostInstall {
				return fmt.Errorf("the action must be PostInstall, When nodes is specified with a label")
			}
			client, err := k8s.Newk8sClient()
			if err != nil {
				return err
			}
			ipList, err := client.ListNodeIPByLabel(strings.TrimSpace(on))
			if err != nil {
				return err
			}
			if len(ipList) == 0 {
				return fmt.Errorf("nodes is not found by label [%s]", on)
			}
			allHostIP = ipList
		} else if on == common.MASTER || on == common.NODE {
			allHostIP = context.Cluster.GetIPSByRole(on)
		} else {
			allHostIP = utils.DisassembleIPList(on)
		}
	}
	if len(allHostIP) == 0 {
		logger.Info("not found ip list with cluster role %s", context.Plugin.Spec.On)
		return nil
	}

	err := l.formatData(context.Plugin.Spec.Data)
	if err != nil {
		return fmt.Errorf("failed to format data from %s: %v", context.Plugin.Spec.Data, err)
	}

	k8sClient, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}

	nodeList, err := k8sClient.ListNodes()
	if err != nil {
		return err
	}
	for _, n := range nodeList.Items {
		node := n
		for _, v := range node.Status.Addresses {
			if !utils.InList(v.Address, allHostIP) {
				continue
			}
			updateTaints := l.UpdateTaints(node.Spec.Taints)
			if updateTaints != nil {
				node.Spec.Taints = updateTaints
				_, err := k8sClient.UpdateNode(node)
				if err != nil {
					return err
				}
			}
			break
		}
	}

	return nil
}

//key1=value1:NoSchedule;key1=value1:NoSchedule-;key1:NoSchedule;key1:NoSchedule-;key1=:NoSchedule-;key1=value1:NoSchedule
func (l *Taint) formatData(data string) error {
	items := strings.Split(data, "\n")
	for _, v := range items {
		v = strings.TrimSpace(v)
		if strings.HasSuffix(v, "#") || v == "" {
			continue
		}
		isDelTaint := false
		if strings.HasSuffix(v, DelSymbol) {
			isDelTaint = true
		}
		taintArgs := strings.Split(v, ColonSymbol)
		if len(taintArgs) != 2 && isDelTaint {
			return fmt.Errorf("error: invalid taint data: %s", v)
		}
		kv, effect := taintArgs[0], taintArgs[1]
		effect = strings.TrimSuffix(effect, DelSymbol)
		if NotInEffect(v1.TaintEffect(effect), TaintEffectValues) && effect != "" {
			return fmt.Errorf("taint effect %s need in %v", v, TaintEffectValues)
		}
		kvList := strings.Split(kv, EqualSymbol)
		key, value := kvList[0], ""
		if len(kvList) > 2 || key == "" {
			return fmt.Errorf("error: invalid taint data: %s", v)
		}
		if len(kvList) == 2 {
			value = kvList[1]
		}
		taint := newTaintStruct(key, value, effect)
		if isDelTaint {
			l.DelTaintList = append(l.DelTaintList, taint)
			continue
		}
		l.AddTaintList = append(l.AddTaintList, taint)
	}
	return nil
}

// UpdateTaints return nil mean's needn't update taint
func (l *Taint) UpdateTaints(taints []v1.Taint) []v1.Taint {
	needUpdate := false
	for k, v := range taints {
		l.removePresenceTaint(v)
		if l.isDelTaint(v) {
			needUpdate = true
			taints = append(taints[:k], taints[k+1:]...)
		}
	}
	if len(l.AddTaintList) == 0 && needUpdate {
		return nil
	}
	return append(taints, l.AddTaintList...)
}

//Remove existing taint
func (l *Taint) removePresenceTaint(taint v1.Taint) {
	for k, v := range l.AddTaintList {
		if v.Key == taint.Key && v.Value == taint.Value && v.Effect == taint.Effect {
			logger.Info("taint %s already exist", l.AddTaintList[k].String())
			l.AddTaintList = append(l.AddTaintList[:k], l.AddTaintList[k+1:]...)
		}
	}
}

func (l *Taint) isDelTaint(taint v1.Taint) bool {
	for _, v := range l.DelTaintList {
		if v.Key == taint.Key && (v.Effect == taint.Effect || v.Effect == "") {
			return true
		}
	}
	return false
}

func NotInEffect(effect v1.TaintEffect, effects []v1.TaintEffect) bool {
	for _, e := range effects {
		if e == effect {
			return false
		}
	}
	return true
}
