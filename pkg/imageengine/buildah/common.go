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

	"github.com/containers/buildah"

	"os"
	"time"

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/umask"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/pkg/errors"
)

const (
	maxPullPushRetries = 3
	pullPushRetryDelay = 2 * time.Second
)

// TODO find a package to place these flags
const (
	OCIManifestDir  = "oci-dir"
	OCIArchive      = "oci-archive"
	V2s2ManifestDir = "docker-dir"
	V2s2Archive     = "docker-archive"
)

func getStore(configurations *options.EngineGlobalConfigurations) (storage.Store, error) {
	options, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
	if err != nil {
		return nil, err
	}

	if configurations != nil {
		if len(configurations.GraphRoot) > 0 {
			options.GraphRoot = configurations.GraphRoot
		}
		if len(configurations.RunRoot) > 0 {
			options.RunRoot = configurations.RunRoot
		}
	}

	// Do not allow to mount a graphdriver that is not vfs if we are creating the userns as part
	// of the mount command.
	// Differently, allow the mount if we are already in a userns, as the mount point will still
	// be accessible once "buildah mount" exits.
	if os.Geteuid() != 0 && options.GraphDriverName != "vfs" {
		return nil, errors.Errorf("cannot mount using driver %s in rootless mode. You need to run it in a `buildah unshare` session", options.GraphDriverName)
	}

	umask.Check()

	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	return store, err
}

func OpenBuilder(ctx context.Context, store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(errors.Cause(err)) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(ctx, store, options)
		}
	}
	if err != nil {
		return nil, err
	}
	if builder == nil {
		return nil, errors.Errorf("error finding build container")
	}
	return builder, nil
}

func openImage(ctx context.Context, sc *types.SystemContext, store storage.Store, name string) (builder *buildah.Builder, err error) {
	options := buildah.ImportFromImageOptions{
		Image:         name,
		SystemContext: sc,
	}
	builder, err = buildah.ImportBuilderFromImage(ctx, store, options)
	if err != nil {
		return nil, err
	}
	if builder == nil {
		return nil, errors.Errorf("error mocking up build configuration")
	}
	return builder, nil
}

// getContext returns a context.TODO
func getContext() context.Context {
	return context.TODO()
}

func getImageType(format string) (string, error) {
	switch format {
	case define.OCI:
		return define.OCIv1ImageManifest, nil
	case define.DOCKER:
		return define.Dockerv2ImageManifest, nil
	default:
		return "", errors.Errorf("unrecognized image type %q", format)
	}
}

func defaultIsolationOption() (define.Isolation, error) {
	return parse.IsolationOption("")
}

func defaultNamespaceOptions() (namespaceOptions define.NamespaceOptions, networkPolicy define.NetworkConfigurationPolicy) {
	options := make(define.NamespaceOptions, 0, 7)
	policy := define.NetworkEnabled
	options.AddOrReplace(define.NamespaceOption{
		Name: "network",
		Host: true,
	})

	return options, policy
}
