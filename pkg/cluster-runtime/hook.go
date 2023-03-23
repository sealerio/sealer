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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/infradriver"
	v1 "github.com/sealerio/sealer/types/api/v1"
	netUtils "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/yaml"
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
	//UpgradeCluster on master0
	UpgradeCluster Phase = "upgrade"
	//RollbackCluster on master0
	RollbackCluster Phase = "rollback"

	//PreInitHost on role
	PreInitHost Phase = "pre-init-host"
	//PostInitHost on role
	PostInitHost Phase = "post-init-host"
	//PreCleanHost on role
	PreCleanHost Phase = "pre-clean-host"
	//PostCleanHost on role
	PostCleanHost Phase = "post-clean-host"
	//UpgradeHost on role
	UpgradeHost Phase = "upgrade-host"
)

type HookType string

type Scope string

type Phase string

const (
	ExtraOptionSkipWhenWorkspaceNotExists = "SkipWhenWorkspaceNotExists"
)

type HookFunc func(data string, onHosts []net.IP, driver infradriver.InfraDriver, extraOpts map[string]bool) error

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
		logrus.Debugf("no hooks found at phase: %s", phase)
		return nil
	}

	extraOpts := map[string]bool{}
	if phase == PostCleanHost || phase == PreCleanHost {
		extraOpts[ExtraOptionSkipWhenWorkspaceNotExists] = true
	}

	// sorted by hookConfig name in alphabetical order
	sort.Sort(hookConfigList)
	for _, hookConfig := range hookConfigList {
		var targetHosts []net.IP
		expectedHosts := i.getHostIPListByScope(hookConfig.Scope)
		// Make sure each host got from Scope is in the given host ip list.
		for _, expected := range expectedHosts {
			if netUtils.IsInIPList(expected, hosts) {
				targetHosts = append(targetHosts, expected)
			}
		}

		if len(targetHosts) == 0 {
			logrus.Debugf("no expected host found from hook %s", hookConfig.Name)
			continue
		}

		logrus.Infof("start to run hook(%s) on host(%s)", hookConfig.Name, targetHosts)
		if err := hookFactories[hookConfig.Type](hookConfig.Data, targetHosts, i.infraDriver, extraOpts); err != nil {
			return fmt.Errorf("failed to run hook: %s", hookConfig.Name)
		}
	}

	return nil
}

// runClusterHook run cluster scope hook by Phase that means will only execute hook on master0.
func (i *Installer) runClusterHook(master0 net.IP, phase Phase) error {
	hookConfigList, ok := i.hooks[phase]
	if !ok {
		logrus.Debugf("no hooks found at phase: %s", phase)
		return nil
	}
	// sorted by hookConfig name in alphabetical order
	sort.Sort(hookConfigList)

	extraOpts := map[string]bool{}
	if phase == PreUnInstallCluster || phase == PostUnInstallCluster {
		extraOpts[ExtraOptionSkipWhenWorkspaceNotExists] = true
	}

	for _, hookConfig := range hookConfigList {
		logrus.Infof("start to run hook(%s) on host(%s)", hookConfig.Name, master0)
		if err := hookFactories[hookConfig.Type](hookConfig.Data, []net.IP{master0}, i.infraDriver, extraOpts); err != nil {
			return fmt.Errorf("failed to run hook: %s", hookConfig.Name)
		}
	}

	return nil
}

// getHostIPListByScope get ip list for scope, support use '|' to specify multiple scopes, they are ORed
func (i *Installer) getHostIPListByScope(scope Scope) []net.IP {
	var ret []net.IP
	scopes := strings.Split(string(scope), "|")
	for _, s := range scopes {
		hosts := i.infraDriver.GetHostIPListByRole(strings.TrimSpace(s))

		// remove duplicates
		for _, h := range hosts {
			if !netUtils.IsInIPList(h, ret) {
				ret = append(ret, h)
			}
		}
	}

	return ret
}

func NewShellHook() HookFunc {
	return func(cmd string, hosts []net.IP, driver infradriver.InfraDriver, extraOpts map[string]bool) error {
		rootfs := driver.GetClusterRootfsPath()
		for _, ip := range hosts {
			logrus.Infof("start to run hook on host %s", ip.String())
			wrappedCmd := fmt.Sprintf(common.CdAndExecCmd, rootfs, cmd)
			if extraOpts[ExtraOptionSkipWhenWorkspaceNotExists] {
				wrappedCmd = fmt.Sprintf(common.CdIfExistAndExecCmd, rootfs, rootfs, cmd)
			}

			err := driver.CmdAsync(ip, driver.GetHostEnv(ip), wrappedCmd)
			if err != nil {
				return fmt.Errorf("failed to run shell hook(%s) on host(%s): %v", wrappedCmd, ip.String(), err)
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

func transferPluginsToHooks(plugins []v1.Plugin) (map[Phase]HookConfigList, error) {
	hooks := make(map[Phase]HookConfigList)

	for _, pluginConfig := range plugins {
		pluginConfig.Spec.Data = strings.TrimSuffix(pluginConfig.Spec.Data, "\n")
		hookType := HookType(pluginConfig.Spec.Type)

		_, ok := hookFactories[hookType]
		if !ok {
			return nil, fmt.Errorf("hook type: %s is not registered", hookType)
		}

		//split pluginConfig.Spec.Action with "|" to support combined actions
		phaseList := strings.Split(pluginConfig.Spec.Action, "|")
		for _, phase := range phaseList {
			if phase == "" {
				continue
			}
			hookConfig := HookConfig{
				Name:  pluginConfig.Name,
				Data:  pluginConfig.Spec.Data,
				Type:  hookType,
				Phase: Phase(phase),
				Scope: Scope(pluginConfig.Spec.Scope),
			}

			if _, ok = hooks[hookConfig.Phase]; !ok {
				// add new Phase
				hooks[hookConfig.Phase] = []HookConfig{hookConfig}
			} else {
				hooks[hookConfig.Phase] = append(hooks[hookConfig.Phase], hookConfig)
			}
		}
	}
	return hooks, nil
}

// LoadPluginsFromFile load plugin config files from $rootfs/plugins dir.
func LoadPluginsFromFile(pluginPath string) ([]v1.Plugin, error) {
	_, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadDir plugin dir %s: %v", pluginPath, err)
	}

	var plugins []v1.Plugin
	for _, f := range files {
		if !yaml.Matcher(f.Name()) {
			continue
		}
		pluginFile := filepath.Join(pluginPath, f.Name())
		pluginList, err := decodePluginFile(pluginFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode plugin file %s: %v", pluginFile, err)
		}
		plugins = append(plugins, pluginList...)
	}

	return plugins, nil
}

func decodePluginFile(pluginFile string) ([]v1.Plugin, error) {
	var plugins []v1.Plugin
	data, err := os.ReadFile(filepath.Clean(pluginFile))
	if err != nil {
		return nil, err
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bufio.NewReaderSize(bytes.NewReader(data), 4096))
	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				return plugins, nil
			}
			return nil, err
		}

		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		if err := k8syaml.Unmarshal(ext.Raw, &metaType); err != nil {
			return nil, fmt.Errorf("failed to decode TypeMeta: %v", err)
		}

		var plu v1.Plugin
		if err := k8syaml.Unmarshal(ext.Raw, &plu); err != nil {
			return nil, fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
		}

		plu.Spec.Data = strings.TrimSuffix(plu.Spec.Data, "\n")
		plugins = append(plugins, plu)
	}
}

func init() {
	Register(ShellHook, NewShellHook())
}
