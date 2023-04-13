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

package imageengine

import (
	"github.com/containers/common/libimage"
	"github.com/opencontainers/go-digest"

	v1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"
)

type Interface interface {
	Build(sealerBuildFlags *options.BuildOptions) (string, error)

	CreateContainer(opts *options.FromOptions) (string, error)

	CreateWorkingContainer(opts *options.BuildRootfsOptions) (string, error)

	Mount(opts *options.MountOptions) ([]options.JSONMount, error)

	Copy(opts *options.CopyOptions) error

	Commit(opts *options.CommitOptions) (string, error)

	Config(opts *options.ConfigOptions) error

	Login(opts *options.LoginOptions) error

	Logout(opts *options.LogoutOptions) error

	Push(opts *options.PushOptions) error

	Pull(opts *options.PullOptions) (string, error)

	Images(opts *options.ImagesOptions) error

	Save(opts *options.SaveOptions) error

	Load(opts *options.LoadOptions) error

	RemoveImage(opts *options.RemoveImageOptions) error

	RemoveContainer(opts *options.RemoveContainerOptions) error

	Tag(opts *options.TagOptions) error

	Inspect(opts *options.InspectOptions) (*v1.ImageSpec, error)

	LookupManifest(name string) (*libimage.ManifestList, error)

	CreateManifest(name string, opts *options.ManifestCreateOpts) (string, error)

	DeleteManifests(names []string, opts *options.ManifestDeleteOpts) error

	InspectManifest(name string, opts *options.ManifestInspectOpts) (*libimage.ManifestListData, error)

	PushManifest(name, destSpec string, opts *options.PushOptions) error

	AddToManifest(name string, imageNameOrIDList []string, opts *options.ManifestAddOpts) error

	RemoveFromManifest(name string, instanceDigest digest.Digest, opts *options.ManifestRemoveOpts) error
}
