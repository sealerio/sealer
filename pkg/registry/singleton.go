// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:19 PM
// @File : local
//

package registry

import (
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
)

type localSingletonConfigurator struct {
	LocalRegistry
	ContainerRuntimeInfo containerruntime.Info
}

func (c *localSingletonConfigurator) Reconcile() (Driver, error) {

}
