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

package docker

import (
	"crypto/sha1" // #nosec
	"encoding/binary"
	"net"

	"github.com/docker/docker/api/types/mount"
)

func DefaultMounts() []mount.Mount {
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/lib/modules",
			Target:   "/lib/modules",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		},
		{
			Type:     mount.TypeVolume,
			Source:   "",
			Target:   "/var",
			ReadOnly: false,
			VolumeOptions: &mount.VolumeOptions{
				DriverConfig: &mount.Driver{
					Name: "local",
				},
			},
		},
		{
			Type:     mount.TypeTmpfs,
			Source:   "",
			Target:   "/tmp",
			ReadOnly: false,
		},
		{
			Type:     mount.TypeTmpfs,
			Source:   "",
			Target:   "/run",
			ReadOnly: false,
		},
	}
	return mounts
}

func GenerateSubnetFromName(name string, attempt int32) string {
	ip := make([]byte, 16)
	ip[0] = 0xfc
	ip[1] = 0x00
	h := sha1.New() // #nosec
	_, _ = h.Write([]byte(name))
	_ = binary.Write(h, binary.LittleEndian, attempt)
	bs := h.Sum(nil)
	for i := 2; i < 8; i++ {
		ip[i] = bs[i]
	}
	subnet := &net.IPNet{
		IP:   net.IP(ip),
		Mask: net.CIDRMask(64, 128),
	}
	return subnet.String()
}
