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
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/util"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sirupsen/logrus"
)

func (engine *Engine) Push(opts *options.PushOptions) error {
	if len(opts.Image) == 0 {
		return errors.New("At least a source image ID must be specified")
	}
	systemCxt := engine.SystemContext()
	if err := auth.CheckAuthFile(systemCxt.AuthFilePath); err != nil {
		return err
	}
	systemCxt.OCIInsecureSkipTLSVerify = opts.SkipTLSVerify
	systemCxt.DockerInsecureSkipTLSVerify = types.NewOptionalBool(opts.SkipTLSVerify)

	src, destSpec := opts.Image, opts.Image
	if len(opts.Destination) != 0 {
		destSpec = opts.Destination
	}
	compress := define.Gzip
	store := engine.ImageStore()
	dest, err := alltransports.ParseImageName(destSpec)
	// add the docker:// transport to see if they neglected it.
	if err != nil {
		destTransport := strings.Split(destSpec, ":")[0]
		if t := transports.Get(destTransport); t != nil {
			return err
		}

		if strings.Contains(destSpec, "://") {
			return err
		}

		destSpec = "docker://" + destSpec
		dest2, err2 := alltransports.ParseImageName(destSpec)
		if err2 != nil {
			return err
		}
		dest = dest2
		logrus.Debugf("Assuming docker:// as the transport method for DESTINATION: %s", destSpec)
	}

	img, _, err := engine.ImageRuntime().LookupImage(src, &libimage.LookupImageOptions{
		ManifestList: true,
	})
	if err != nil {
		return err
	}

	isManifest, err := img.IsManifestList(getContext())
	if err != nil {
		return err
	}

	if isManifest {
		if manifestsErr := engine.PushManifest(src, destSpec, opts); manifestsErr == nil {
			return nil
		}
		return util.GetFailureCause(err, errors.Wrapf(err, "error pushing image %q to %q", src, destSpec))
	}

	var manifestType string
	if opts.Format != "" {
		switch opts.Format {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s1":
			manifestType = manifest.DockerV2Schema1SignedMediaType
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return errors.Errorf("unknown format %q. Choose one of the supported formats: 'oci', 'v2s1', or 'v2s2'", opts.Format)
		}
	}

	options := buildah.PushOptions{
		Compression:   compress,
		ManifestType:  manifestType,
		Store:         store,
		SystemContext: systemCxt,
		MaxRetries:    maxPullPushRetries,
		RetryDelay:    pullPushRetryDelay,
	}
	if !opts.Quiet {
		options.ReportWriter = os.Stderr
	}
	ref, digest, err := buildah.Push(getContext(), src, dest, options)
	if err != nil {
		return util.GetFailureCause(err, errors.Wrapf(err, "error pushing image %q to %q", src, destSpec))
	}
	if ref != nil {
		logrus.Debugf("pushed image %q with digest %s", ref, digest.String())
	} else {
		logrus.Debugf("pushed image with digest %s", digest.String())
	}

	logrus.Infof("successfully pushed %s with digest %s", transports.ImageName(dest), digest.String())

	return nil
}
