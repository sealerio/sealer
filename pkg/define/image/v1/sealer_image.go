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

package v1

import (
	"encoding/json"

	applicationV1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
	apiv1 "github.com/sealerio/sealer/types/api/v1"

	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	SealerImageExtension          = "sealer.image.extension"
	SealerImageContainerImageList = "sealer.image.container.images"

	// Kube image type
	KubeInstaller = "kube-installer"
	AppInstaller  = "app-installer"

	ImageSpecSchemaVersionV1Beta1 = "v1alpha1"
)

type ImageSpec struct {
	// ID the image id
	ID string `json:"id"`

	// Name the image name
	Name string `json:"name"`

	// Digest the image digest
	Digest string `json:"digest"`

	// ManifestV1 OCI version v1 image manifest
	ManifestV1 ociv1.Manifest `json:"manifestv1"`

	// OCIv1 OCI version v1 image spec
	OCIv1 ociv1.Image `json:"ociv1"`

	ImageExtension ImageExtension `json:"imageExtension"`

	// ContainerImageList the container image list contained in the sealer image
	//
	// TODO: populate this value during the build phase
	ContainerImageList []*ContainerImage `json:"containerImageList,omitempty"`
}

type ContainerImage struct {
	// Image the container image name
	Image string `json:"image" yaml:"image"`

	// AppName the mame of the app to which the container image belongs
	//
	// NOTE: A container image may not belong to any app. In this case, the appName value is null.
	AppName string `json:"appName,omitempty" yaml:"appName"`

	// Platform the container image platform
	Platform *apiv1.Platform `json:"platform" yaml:"platform"`

	// TODO: add more info about container image if necessary such as resourceKind, resourceName, etc.
}

func GetImageSliceFromContainerImageList(containerImageList []*ContainerImage) []string {
	var images []string
	for _, containerImage := range containerImageList {
		images = append(images, containerImage.Image)
	}
	return images
}

// NOTE: the UnmarshalJSON function of ImageExtension has been overrode
type ImageExtension struct {
	// BuildClient build client info
	BuildClient BuildClient `json:"buildClient"`

	// image spec schema version
	SchemaVersion string `json:"schemaVersion"`

	// sealer image type, like AppImage
	Type string `json:"type,omitempty"`

	// applications in the sealer image
	Applications []version.VersionedApplication `json:"applications,omitempty"`

	// Labels are metadata to the sealer image
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Env is a set of key value pair.
	// set to sealer image some default parameters which is in global level.
	// user could overwrite it through v2.ClusterSpec at run stage.
	Env map[string]string `json:"env,omitempty"`

	// launch spec will declare
	Launch Launch `json:"launch,omitempty"`
}

type BuildClient struct {
	SealerVersion string `json:"sealerVersion"`

	BuildahVersion string `json:"buildahVersion"`
}

type Launch struct {
	Cmds []string `json:"cmds,omitempty"`

	// user specified LAUNCH instruction
	AppNames []string `json:"app_names,omitempty"`
}

type v1ImageExtension struct {
	// BuildClient build client info
	BuildClient BuildClient `json:"buildClient"`
	// image spec schema version
	SchemaVersion string `json:"schemaVersion"`
	// sealer image type, like AppImage
	Type string `json:"type,omitempty"`
	// Labels are metadata to the sealer image
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	// applications in the sealer image
	Applications []applicationV1.Application `json:"applications,omitempty"`
	// launch spec will declare
	Launch Launch `json:"launch,omitempty"`
	// Env global env
	Env map[string]string `json:"env,omitempty"`
}

func (ie *ImageExtension) UnmarshalJSON(data []byte) error {
	*ie = ImageExtension{}
	v1Ex := v1ImageExtension{}
	if err := json.Unmarshal(data, &v1Ex); err != nil {
		return err
	}

	(*ie).BuildClient = v1Ex.BuildClient
	(*ie).SchemaVersion = v1Ex.SchemaVersion
	(*ie).Labels = v1Ex.Labels
	(*ie).Env = v1Ex.Env
	(*ie).Type = v1Ex.Type
	(*ie).Applications = make([]version.VersionedApplication, len(v1Ex.Applications))
	for i, app := range v1Ex.Applications {
		tmpApp := app
		(*ie).Applications[i] = &tmpApp
	}
	(*ie).Launch = v1Ex.Launch
	return nil
}
