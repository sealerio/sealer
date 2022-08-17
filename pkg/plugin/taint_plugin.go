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
	"net"
	"strings"

	"github.com/sealerio/sealer/pkg/client/k8s"
	utilsnet "github.com/sealerio/sealer/utils/net"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

var TaintEffectValues = []v1.TaintEffect{v1.TaintEffectNoSchedule, v1.TaintEffectNoExecute, v1.TaintEffectPreferNoSchedule}

type TaintList map[string]*taintList //map[ip]taint

type Taint struct {
	IPList []string
	TaintList
}

type taintList struct {
	DelTaintList []v1.Taint
	AddTaintList []v1.Taint
}

func NewTaintPlugin() Interface {
	return &Taint{TaintList: make(map[string]*taintList)}
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
//  data: 192.168.56.3 key1=value1:NoSchedule ## add taint
//  #data: 192.168.56.3 key1=value1:NoSchedule- ## del taint

func (l *Taint) Run(context Context, phase Phase) (err error) {
	if phase != PhasePreGuest || context.Plugin.Spec.Type != TaintPlugin {
		return nil
	}

	err = l.formatData(context.Plugin.Spec.Data)
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
			if strUtils.NotIn(v.Address, l.IPList) || utilsnet.NotInIPList(net.ParseIP(v.Address), context.Host) {
				continue
			}
			updateTaints := l.UpdateTaints(node.Spec.Taints, v.Address)
			if updateTaints != nil {
				node.Spec.Taints = updateTaints
				_, err := k8sClient.UpdateNode(node)
				if err != nil {
					return err
				}
				logrus.Infof("succeed in updating node(%s) taints to %v", v.Address, updateTaints)
			}
			break
		}
	}

	return nil
}

// key1=value1:NoSchedule;key1=value1:NoSchedule-;key1:NoSchedule;key1:NoSchedule-;key1=:NoSchedule-;key1=value1:NoSchedule
func (l *Taint) formatData(data string) error {
	items := strings.Split(data, "\n")
	if l.TaintList == nil {
		l.TaintList = make(map[string]*taintList)
	}
	for _, v := range items {
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "#") || v == "" {
			continue
		}
		temps := strings.Split(v, " ")
		if len(temps) != 2 {
			return fmt.Errorf("faild to split taint argument: %s", v)
		}
		ips := temps[0]
		ipStr, err := utilsnet.AssemblyIPList(ips)
		if err != nil {
			return err
		}
		l.IPList = append(l.IPList, ipStr)
		//kubectl taint nodes xxx key- : remove all key related taints
		if l.TaintList[ipStr] == nil {
			l.TaintList[ipStr] = &taintList{}
		}
		if strings.HasSuffix(temps[1], DelSymbol) && !strings.Contains(temps[1], ColonSymbol) && !strings.Contains(temps[1], EqualSymbol) {
			l.TaintList[ips].DelTaintList = append(l.TaintList[ips].DelTaintList, newTaintStruct(strings.TrimSuffix(temps[1], DelSymbol), "", ""))
			continue
		}
		taintArgs := strings.Split(temps[1], ColonSymbol)
		if len(taintArgs) != 2 {
			return fmt.Errorf("error: invalid taint data: %s", v)
		}
		kv, effect := taintArgs[0], taintArgs[1]
		effect = strings.TrimSuffix(effect, DelSymbol)
		if NotInEffect(v1.TaintEffect(effect), TaintEffectValues) {
			return fmt.Errorf("taint effect %s need in %v", v, TaintEffectValues)
		}
		kvList := strings.Split(kv, EqualSymbol)
		key, value := kvList[0], ""
		if len(kvList) > 2 || key == "" {
			return fmt.Errorf("error: invalid taint data: %s", temps[1])
		}
		if len(kvList) == 2 {
			value = kvList[1]
		}
		taint := newTaintStruct(key, value, effect)
		if _, ok := l.TaintList[ips]; !ok {
			l.TaintList[ips] = &taintList{}
		}
		if strings.HasSuffix(temps[1], DelSymbol) {
			l.TaintList[ips].DelTaintList = append(l.TaintList[ips].DelTaintList, taint)
			continue
		}
		l.TaintList[ips].AddTaintList = append(l.TaintList[ips].AddTaintList, taint)
	}
	return nil
}

// UpdateTaints return nil mean's needn't update taint
func (l *Taint) UpdateTaints(taints []v1.Taint, ip string) []v1.Taint {
	needDel := false
	updateTaints := []v1.Taint{}
	for k, v := range taints {
		l.removePresenceTaint(v, ip)
		if l.isDelTaint(v, ip) {
			needDel = true
			continue
		}
		updateTaints = append(updateTaints, taints[k])
	}
	if len(l.TaintList[ip].AddTaintList) == 0 && !needDel {
		return nil
	}
	return append(updateTaints, l.TaintList[ip].AddTaintList...)
}

// Remove existing taint
func (l *Taint) removePresenceTaint(taint v1.Taint, ip string) {
	for k, v := range l.TaintList[ip].AddTaintList {
		if v.Key == taint.Key && v.Value == taint.Value && v.Effect == taint.Effect {
			logrus.Infof("taint %s already exist", l.TaintList[ip].AddTaintList[k].String())
			l.TaintList[ip].AddTaintList = append(l.TaintList[ip].AddTaintList[:k], l.TaintList[ip].AddTaintList[k+1:]...)
			break
		}
	}
}

func (l *Taint) isDelTaint(taint v1.Taint, ip string) bool {
	for _, v := range l.TaintList[ip].DelTaintList {
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
