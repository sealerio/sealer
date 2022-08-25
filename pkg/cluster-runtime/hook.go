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

package cluster_runtime

import (
	"fmt"
	"net"
)

// HookConfig tell us how to configure hooks for cluster
type HookConfig struct{}

type Phase string
type HookFunc func() error

const (
	PreInstall    Phase = "pre-install"
	PostInstall   Phase = "post-install"
	PreUnInstall  Phase = "pre-uninstall"
	PostUnInstall Phase = "post-uninstall"
	PreScaleUp    Phase = "pre-scaleup"
	PostScaleUp   Phase = "post-scaleup"

	PreInitHost   Phase = "pre-init-host"
	PostInitHost  Phase = "post-init-host"
	PreCleanHost  Phase = "pre-clean-host"
	PostCleanHost Phase = "post-clean-host"
)

func (i *Installer) AddHook(phase Phase, hook HookFunc) error {
	if i.hooks == nil {
		i.hooks = map[Phase][]HookFunc{}
	}

	i.hooks[phase] = append(i.hooks[phase], hook)

	return nil
}

// run hooks
func (i *Installer) runHook(phase Phase) error {
	hookList, ok := i.hooks[phase]
	if !ok {
		return fmt.Errorf("failed to load hook: %s", phase)
	}

	for _, hook := range hookList {
		if err := hook(); err != nil {
			return fmt.Errorf("failed to run hook: %s", phase)
		}
	}

	return nil
}

// run hooks
func (i *Installer) runHookOnHosts(phase Phase, hosts []net.IP) error {
	return nil
}
