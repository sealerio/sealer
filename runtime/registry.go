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

package runtime

import (
	"fmt"
)

func getRegistryHost(ip string) (host string) {
	return fmt.Sprintf("%s %s", ip, SeaHub)
}

const registryName = "sealer-registry"

//Only use this for join and init, due to the initiation operations
func (d *Default) EnsureRegistryOnMaster0() error {
	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", d.Rootfs, d.Rootfs)
	return d.SSH.CmdAsync(d.Masters[0], cmd)
}

func (d *Default) RecycleRegistryOnMaster0() error {
	cmd := fmt.Sprintf("docker stop %s || true && docker rm %s || true", registryName, registryName)
	return d.SSH.CmdAsync(d.Masters[0], cmd)
}

func (d *Default) EnsureRegistryHost() error {
	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", d.Rootfs, d.Rootfs)
	return d.SSH.CmdAsync(d.RegistryHost, cmd)
}

func (d *Default) RecycleRegistryHosst() error {
	cmd := fmt.Sprintf("docker stop %s || true && docker rm %s || true", registryName, registryName)
	return d.SSH.CmdAsync(d.RegistryHost, cmd)
}
