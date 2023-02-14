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
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/util"
	"github.com/containers/image/v5/pkg/shortnames"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) Commit(opts *options.CommitOptions) (string, error) {
	var dest types.ImageReference
	if len(opts.ContainerID) == 0 {
		return "", errors.Errorf("container ID must be specified")
	}
	if len(opts.Image) == 0 && len(opts.Manifest) == 0 {
		return "", errors.Errorf("image name should be specified")
	}

	name := opts.ContainerID
	image := opts.Image
	manifest := opts.Manifest
	compress := define.Gzip
	if opts.DisableCompression {
		compress = define.Uncompressed
	}

	format, err := getImageType(opts.Format)
	if err != nil {
		return "", err
	}

	ctx := getContext()
	store := engine.ImageStore()
	builder, err := OpenBuilder(ctx, store, name)
	if err != nil {
		return "", errors.Wrapf(err, "error reading build container %q", name)
	}

	systemCxt := engine.SystemContext()

	// If the user specified an image, we may need to massage it a bit if
	// no transport is specified.
	// TODO we support commit to local image only, we'd better limit the input of name
	if image != "" {
		dest, err = getImageReference(image, store, systemCxt)
		if err != nil {
			return "", err
		}
	}

	options := buildah.CommitOptions{
		PreferredManifestType: format,
		Manifest:              manifest,
		Compression:           compress,
		SystemContext:         systemCxt,
		Squash:                opts.Squash,
	}
	if opts.Timestamp != 0 {
		timestamp := time.Unix(opts.Timestamp, 0).UTC()
		options.HistoryTimestamp = &timestamp
	}

	if !opts.Quiet {
		options.ReportWriter = os.Stderr
	}
	id, ref, _, err := builder.Commit(ctx, dest, options)
	if err != nil {
		return "", util.GetFailureCause(err, errors.Wrapf(err, "error committing container %q to %q", builder.Container, image))
	}
	if ref != nil && id != "" {
		logrus.Debugf("wrote image %s with ID %s", ref, id)
	} else if ref != nil {
		logrus.Debugf("wrote image %s", ref)
	} else if id != "" {
		logrus.Debugf("wrote image with ID %s", id)
	} else {
		logrus.Debugf("wrote image")
	}

	if opts.Rm {
		return id, builder.Delete()
	}
	return id, nil
}

func getImageReference(image string, store storage.Store, systemCxt *types.SystemContext) (types.ImageReference, error) {
	dest, err := alltransports.ParseImageName(image)
	if err == nil {
		return dest, nil
	}

	candidates, err := shortnames.ResolveLocally(systemCxt, image)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errors.Errorf("no candidate tags for target image name %q", image)
	}
	dest, err = storageTransport.Transport.ParseStoreReference(store, candidates[0].String())
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing target image name %q", image)
	}

	return dest, nil
}
