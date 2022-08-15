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
	"context"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/common/libimage"
	"github.com/pkg/errors"
)

func (engine *Engine) Save(opts *options.SaveOptions) error {
	if len(opts.ImageNameOrID) == 0 {
		return errors.New("image name or id must be specified")
	}
	if opts.Compress && (opts.Format != OCIManifestDir && opts.Format != V2s2ManifestDir) {
		return errors.New("--compress can only be set when --format is either 'oci-dir' or 'docker-dir'")
	}

	saveOptions := &libimage.SaveOptions{
		CopyOptions: libimage.CopyOptions{
			DirForceCompress:            opts.Compress,
			OciAcceptUncompressedLayers: false,
			// Force signature removal to preserve backwards compat.
			// See https://github.com/containers/podman/pull/11669#issuecomment-925250264
			RemoveSignatures: true,
		},
	}

	// TODO we can support multiAchieve in the future
	// check podman save
	names := []string{opts.ImageNameOrID}

	return engine.ImageRuntime().Save(context.Background(), names, opts.Format, opts.Output, saveOptions)
}
