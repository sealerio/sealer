// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:18 PM
// @File : driver
//

package registry

// Registry驱动器接口，供其他服务操作Registry
type Driver interface {
	UploadContainerImages2Registry() error
	GetInfo() Info
}

type Info struct {
}
