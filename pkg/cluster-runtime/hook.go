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

package clusterruntime

import (
	"fmt"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/infradriver"
	netUtils "github.com/sealerio/sealer/utils/net"
	"github.com/sirupsen/logrus"
	"net"
	"sort"
)

const (
	ShellHook HookType = "SHELL"
)

const (
	//PreInstallCluster on master0
	PreInstallCluster Phase = "pre-install"
	//PostInstallCluster on master0
	PostInstallCluster Phase = "post-install"
	//PreUnInstallCluster on master0
	PreUnInstallCluster Phase = "pre-uninstall"
	//PostUnInstallCluster on master0
	PostUnInstallCluster Phase = "post-uninstall"
	//PreScaleUpCluster on master0
	PreScaleUpCluster Phase = "pre-scaleup"
	//PostScaleUpCluster on master0
	PostScaleUpCluster Phase = "post-scaleup"

	//PreInitHost on role
	PreInitHost Phase = "pre-init-host"
	//PostInitHost on role
	PostInitHost Phase = "post-init-host"
	//PreCleanHost on role
	PreCleanHost Phase = "pre-clean-host"
	//PostCleanHost on role
	PostCleanHost Phase = "post-clean-host"
)

type HookType string

type Scope string

type Phase string

type HookFunc func(data string, onHosts []net.IP, driver infradriver.InfraDriver) error

var hookFactories = make(map[HookType]HookFunc)

// HookConfig tell us how to configure hooks for cluster
type HookConfig struct {
	// Name defines hook names, will run hooks in alphabetical order.
	Name string `json:"name,omitempty"`
	//Type defines different hook type, currently only have "SHELL","HOSTNAME".
	Type HookType `json:"type,omitempty"`
	// Data real hooks data will be applied at install process.
	Data string `json:"data,omitempty"`
	// Phase defines when to run hooks.
	Phase Phase `json:"Phase,omitempty"`
	// Scope defines which roles of node will be applied with hook Data
	Scope Scope `json:"scope,omitempty"`
}

type HookConfigList []HookConfig

func (r HookConfigList) Len() int           { return len(r) }
func (r HookConfigList) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r HookConfigList) Less(i, j int) bool { return r[i].Name < r[j].Name }

// runHostHook run host scope hook by Phase and only execute hook on the given host list.
func (i *Installer) runHostHook(phase Phase, hosts []net.IP) error {
	hookConfigList, ok := i.hooks[phase]
	if !ok {
		return fmt.Errorf("failed to load hook at phase: %s", phase)
	}

	// sorted by hookConfig name in alphabetical order
	sort.Sort(hookConfigList)
	for _, hookConfig := range hookConfigList {
		var targetHosts []net.IP
		expectedHosts := i.infraDriver.GetHostIPListByRole(string(hookConfig.Scope))
		// Make sure each host got from Scope is in the given host ip list.
		for _, expected := range expectedHosts {
			if !netUtils.NotInIPList(expected, hosts) {
				targetHosts = append(targetHosts, expected)
			}
		}

		if len(targetHosts) == 0 {
			logrus.Debugf("no expected host found from hook %s", hookConfig.Name)
			continue
		}

		if err := hookFactories[hookConfig.Type](hookConfig.Data, targetHosts, i.infraDriver); err != nil {
			return fmt.Errorf("failed to run hook: %s", hookConfig.Name)
		}
	}

	return nil
}

// runClusterHook run cluster scope hook by Phase that means will only execute hook on master0.
func (i *Installer) runClusterHook(phase Phase) error {
	hookConfigList, ok := i.hooks[phase]
	if !ok {
		return fmt.Errorf("failed to load hook at phase: %s", phase)
	}

	master0 := i.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	// sorted by hookConfig name in alphabetical order
	sort.Sort(hookConfigList)
	for _, hookConfig := range hookConfigList {
		if err := hookFactories[hookConfig.Type](hookConfig.Data, []net.IP{master0}, i.infraDriver); err != nil {
			return fmt.Errorf("failed to run hook: %s", hookConfig.Name)
		}
	}

	return nil
}

func NewShellHook() HookFunc {
	return func(data string, hosts []net.IP, driver infradriver.InfraDriver) error {
		for _, ip := range hosts {
			err := driver.CmdAsync(ip, data)
			if err != nil {
				return fmt.Errorf("failed to run shell hook on host(%s): %v", ip.String(), err)
			}
		}

		return nil
	}
}

// Register different hook type with its HookFunc to hookFactories
func Register(name HookType, factory HookFunc) {
	if factory == nil {
		panic("Must not provide nil hookFactory")
	}
	_, registered := hookFactories[name]
	if registered {
		panic(fmt.Sprintf("hookFactory named %s already registered", name))
	}

	hookFactories[name] = factory
}

func init() {
	Register(ShellHook, NewShellHook())
}
