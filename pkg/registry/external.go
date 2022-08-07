// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:19 PM
// @File : external
//

package registry

type externalConfigurator struct {
	Registry
}

func (c *externalConfigurator) Reconcile() (Driver, error) {

}
