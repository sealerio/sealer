// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:03 PM
// @File : driver
//

package cluster_runtime

import "github.com/sealerio/sealer/pkg/client/k8s"

// Kube运行时驱动器接口，供其他服务操作K8s
type KubeRuntimeDriver interface {
	GetClient() (k8s.Client, error)
	ExecWithAdminKubeconfig(Cmds []string) error
}

// Registry驱动器接口，供其他服务操作Registry
type RegistryDriver interface {
	UploadContainerImages2Registry() error
}

// k8s驱动器，类比操作系统接口
type Driver struct {
	RegistryDriver
	KubeRuntimeDriver
}
