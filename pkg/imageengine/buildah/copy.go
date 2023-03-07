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

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah"

	"time"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
)

// Copy will copy files in the host to the container.
// this is a basic ability, but not used in sealer now.
func (engine *Engine) Copy(opts *options.CopyOptions) error {
	if len(opts.Container) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if len(opts.SourcesRel2CxtDir) == 0 {
		return errors.Errorf("src must be specified")
	}
	if len(opts.DestinationInContainer) == 0 {
		return errors.Errorf("destination in container must be specified")
	}

	name := opts.Container
	dest := opts.DestinationInContainer
	store := engine.ImageStore()

	var idMappingOptions *buildah.IDMappingOptions
	contextdir := opts.ContextDir
	if opts.IgnoreFile != "" && contextdir == "" {
		return errors.Errorf("--ignorefile option requires that you specify a context dir using --contextdir")
	}

	builder, err := OpenBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	builder.ContentDigester.Restart()

	options := buildah.AddAndCopyOptions{
		ContextDir:       contextdir,
		IDMappingOptions: idMappingOptions,
	}
	if opts.ContextDir != "" {
		var excludes []string

		excludes, options.IgnoreFile, err = parse.ContainerIgnoreFile(options.ContextDir, opts.IgnoreFile)
		if err != nil {
			return err
		}
		options.Excludes = excludes
	}

	err = builder.Add(dest, false, options, opts.SourcesRel2CxtDir...)
	if err != nil {
		return errors.Wrapf(err, "error adding content to container %q", builder.Container)
	}

	contentType, digest := builder.ContentDigester.Digest()
	if !opts.Quiet {
		logrus.Infof("%s", digest.Hex())
	}
	if contentType != "" {
		contentType = contentType + ":"
	}
	conditionallyAddHistory(builder, opts, "/bin/sh -c #(nop) %s %s%s", "COPY", contentType, digest.Hex())
	return builder.Save()
}

func conditionallyAddHistory(builder *buildah.Builder, opts *options.CopyOptions, createdByFmt string, args ...interface{}) {
	if opts.AddHistory || buildahcli.DefaultHistory() {
		now := time.Now().UTC()
		created := &now
		createdBy := fmt.Sprintf(createdByFmt, args...)
		builder.AddPrependedEmptyLayer(created, createdBy, "", "")
	}
}
