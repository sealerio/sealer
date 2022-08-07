// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:48 PM
// @File : hook
//

package cluster_runtime

import "net"

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
	return nil
}

// run hooks
func (i *Installer) runHookOnHosts(phase Phase, hosts []net.IP) error {
	return nil
}
