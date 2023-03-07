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

	"os"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/pkg/errors"
)

func (engine *Engine) Mount(opts *options.MountOptions) ([]options.JSONMount, error) {
	containers := opts.Containers
	if len(containers) == 0 {
		return []options.JSONMount{}, errors.Errorf("id/name of containers mube be specified")
	}

	store := engine.ImageStore()
	var jsonMounts []options.JSONMount
	var lastError error
	// Do not allow to mount a graphdriver that is not vfs if we are creating the userns as part
	// of the mount command.
	// Differently, allow the mount if we are already in a userns, as the mount point will still
	// be accessible once "buildah mount" exits.
	if os.Geteuid() != 0 && store.GraphDriverName() != "vfs" {
		return []options.JSONMount{}, errors.Errorf("cannot mount using driver %s in rootless mode. You need to run it in a `buildah unshare` session", store.GraphDriverName())
	}

	for _, name := range containers {
		builder, err := OpenBuilder(getContext(), store, name)
		if err != nil {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			lastError = errors.Wrapf(err, "error reading build container %q", name)
			continue
		}
		mountPoint, err := builder.Mount(builder.MountLabel)
		if err != nil {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			lastError = errors.Wrapf(err, "error mounting %q container %q", name, builder.Container)
			continue
		}
		if len(containers) > 1 {
			jsonMounts = append(jsonMounts, options.JSONMount{Container: name, MountPoint: mountPoint})
			continue
		} else {
			jsonMounts = append(jsonMounts, options.JSONMount{MountPoint: mountPoint})
			continue
		}
	}

	return jsonMounts, nil
}
