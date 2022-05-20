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

package distributionutil

import (
	"github.com/sealerio/sealer/pkg/image/types"
	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/manifestlist"
	"github.com/pkg/errors"
)

// PlatformSpecFromOCI creates a platform spec from OCI platform
func PlatformSpecFromOCI(p *v1.Platform) *manifestlist.PlatformSpec {
	if p == nil {
		return nil
	}
	return &manifestlist.PlatformSpec{
		Architecture: p.Architecture,
		OS:           p.OS,
		OSVersion:    p.OSVersion,
		Variant:      p.Variant,
	}
}

func buildManifestDescriptor(descriptor distribution.Descriptor, imageManifest *types.ManifestDescriptor) (manifestlist.ManifestDescriptor, error) {
	manifest := manifestlist.ManifestDescriptor{
		Descriptor: descriptor,
	}

	platform := PlatformSpecFromOCI(&imageManifest.Platform)
	if platform != nil {
		manifest.Platform = *platform
	}

	if err := manifest.Descriptor.Digest.Validate(); err != nil {
		return manifestlist.ManifestDescriptor{}, errors.Wrap(err, "digest parse of image failed")
	}

	return manifest, nil
}
