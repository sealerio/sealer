// alibaba-inc.com Inc.
// Copyright (c) 2004-2021 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2021/2/18 8:03 下午
// @File : registry
//

package runtime

import (
	"fmt"
)

func getRegistryHost(ip string) (host string) {
	return fmt.Sprintf("%s %s", ip, SeaHub)
}

//Only use this for join and init, due to the initiation operations
func (d *Default) EnsureRegistryOnMaster0() error {
	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", d.Rootfs, d.Rootfs)
	return d.SSH.CmdAsync(d.Masters[0], cmd)
}
