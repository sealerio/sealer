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
	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/sealerio/sealer/pkg/define/options"

	"os"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func manifestPush(systemContext *types.SystemContext, store storage.Store, listImageSpec, destSpec string, opts options.PushOptions) error {
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	manifestList, err := runtime.LookupManifestList(listImageSpec)
	if err != nil {
		return err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	dest, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return err
	}

	var manifestType string
	if opts.Format != "" {
		switch opts.Format {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return errors.Errorf("unknown format %q. Choose one of the supported formats: 'oci' or 'v2s2'", opts.Format)
		}
	}

	options := manifests.PushOptions{
		Store:              store,
		SystemContext:      systemContext,
		ImageListSelection: cp.CopySpecificImages,
		Instances:          nil,
		ManifestType:       manifestType,
	}
	if opts.All {
		options.ImageListSelection = cp.CopyAllImages
	}
	if !opts.Quiet {
		options.ReportWriter = os.Stderr
	}

	_, _, err = list.Push(getContext(), dest, options)

	if err == nil && opts.Rm {
		_, err = store.DeleteImage(manifestList.ID(), true)
	}

	return err
}
