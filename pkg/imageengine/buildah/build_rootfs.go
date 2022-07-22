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

package buildah

import (
	"fmt"

	"github.com/sealerio/sealer/pkg/define/options"

	"os"
)

// BuildRootfs will make an image rootfs under /var/lib/containers/storage
// And then link it to the DestDir
// And remember to call RemoveContainer to remove the link and remove the container(umount rootfs) manually.
func (engine *Engine) BuildRootfs(opts *options.BuildRootfsOptions) (containerID string, err error) {
	// TODO clean environment when it fails
	cid, err := engine.CreateContainer(&options.FromOptions{
		Image: opts.ImageNameOrID,
		Quiet: false,
	})
	if err != nil {
		return "", err
	}

	mounts, err := engine.Mount(&options.MountOptions{Containers: []string{cid}})
	if err != nil {
		return "", err
	}

	// remove destination dir if it exists, otherwise the Symlink will fail.
	if _, err = os.Stat(opts.DestDir); err == nil {
		return "", fmt.Errorf("destination directionay %s exists, you should remove it first", opts.DestDir)
	}

	mountPoint := mounts[0].MountPoint
	return cid, os.Symlink(mountPoint, opts.DestDir)
}
