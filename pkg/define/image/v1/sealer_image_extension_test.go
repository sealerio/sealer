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
	"reflect"
	"testing"
	"time"

	"github.com/sealerio/sealer/pkg/define/application"
	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewImageSpec(t *testing.T) {
	type args struct {
		spec *ImageSpec
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				spec: &ImageSpec{
					ID:     "783d1c3814b9cb28dafd7c3ca34b734d5798a61dc523c0d30349636e7cd56cc6",
					Name:   "cloud-image-registry.cn-shanghai.cr.aliyuncs.com/applications/1137465199671599:cnstack-acos-0.0.5-beta-07b8866-70f202",
					Digest: "sha256:ebdd1badbb67f0f3de27317252618759db11782928bd206e9b155d5cc81f2076",
					ManifestV1: ociv1.Manifest{
						Versioned: specs.Versioned{
							SchemaVersion: 2,
						},
						MediaType: "application/vnd.oci.image.manifest.v1+json",
						Config: ociv1.Descriptor{
							MediaType:   "application/vnd.oci.image.config.v1+json",
							Digest:      "sha256:783d1c3814b9cb28dafd7c3ca34b734d5798a61dc523c0d30349636e7cd56cc6",
							Size:        1288,
							URLs:        nil,
							Annotations: nil,
							Platform: &ociv1.Platform{
								Architecture: "",
								OS:           "",
								OSVersion:    "",
								OSFeatures:   nil,
								Variant:      "",
							},
						},
						Layers: []ociv1.Descriptor{
							{
								MediaType:   "application/vnd.oci.image.layer.v1.tar+gzip",
								Digest:      "sha256:4e36d2017e0a3b3e6642471bbcefc8290fc92a173bd3a5e70d9285fe4af6cb51",
								Size:        3077,
								URLs:        nil,
								Annotations: nil,
								Platform: &ociv1.Platform{
									Architecture: "",
									OS:           "",
									OSVersion:    "",
									OSFeatures:   nil,
									Variant:      "",
								},
							},
							{
								MediaType:   "application/vnd.oci.image.layer.v1.tar+gzip",
								Digest:      "sha256:4086b218e01c8713ed47e93f8fd24c2e714a609ff3b5268c91231dd5590e58e4",
								Size:        288,
								URLs:        nil,
								Annotations: nil,
								Platform: &ociv1.Platform{
									Architecture: "",
									OS:           "",
									OSVersion:    "",
									OSFeatures:   nil,
									Variant:      "",
								},
							},
						},
						Annotations: map[string]string{
							"org.opencontainers.image.base.digest": "sha256:d3801a216e40f5bc517b7e8aa1da41c31127f819f38a49356ffbd23a3d476017",
							"sealer.image.extension":               "{\\\"type\\\":\\\"app-installer\\\",\\\"applications\\\":[{\\\"name\\\":\\\"cnstack-acos-0.0.5-beta\\\",\\\"type\\\":\\\"shell\\\",\\\"launchfiles\\\":[\\\"cnstack-acos-0.0.5-beta-install.sh\\\"],\\\"version\\\":\\\"v1\\\"}],\\\"launch\\\":{\\\"cmds\\\":[\\\"bash application/apps/cnstack-acos-0.0.5-beta/cnstack-acos-0.0.5-beta-install.sh\\\"]}}",
						},
					},
					OCIv1: ociv1.Image{
						Created:      wrapTimePoint(time.Now()),
						Author:       "",
						Architecture: "amd64",
						Variant:      "",
						OS:           "linux",
						OSVersion:    "",
						OSFeatures:   nil,
						Config: ociv1.ImageConfig{
							User:         "",
							ExposedPorts: nil,
							Env: []string{
								"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
							},
							Entrypoint: nil,
							Cmd:        nil,
							Volumes:    nil,
							WorkingDir: "",
							Labels: map[string]string{
								"io.buildah.version": "1.27.1",
								"io.sealer.version":  "0.9",
							},
							StopSignal: "",
						},
						RootFS: ociv1.RootFS{
							Type: "layers",
							DiffIDs: []digest.Digest{
								"sha256:aa735d9c504ee5a122bb323c0147a1ad317bd0e72b5eef1094025e793f6f4912",
								"sha256:7c5cf4cbab5b83e603116d3b3f84287df4f8e70c3c35e3cd6d384b451bb8c26d",
								"sha256:e0af0977fcb45ba29be4ef74ec1dcdbb625e01b78905bb089b8cc6b0ffc717ba",
							},
						},
						History: []ociv1.History{
							{
								Created:   wrapTimePoint(time.Now()),
								CreatedBy: "/bin/sh -c #(nop) COPY dir:2127a0f14734aff8fb3128a35824dcbb01dd2d0f8295e55e3ea0ab299daef971 in applications ",
							},
							{
								Created:   wrapTimePoint(time.Now()),
								CreatedBy: "/bin/sh -c #(nop) COPY file:b48c5033638953a52b8c36a41734d2e862787520a3b372e4ef1161d491c7221f in application/apps/cnstack-acos-0.0.5-beta/ ",
								Comment:   "FROM c6dfe3236bb0",
							},
							{
								Created:    wrapTimePoint(time.Now()),
								CreatedBy:  "/bin/sh -c #(nop) LABEL \"io.sealer.version\"=\"0.9\"",
								Comment:    "FROM 4a18afda8bf5",
								EmptyLayer: true,
							},
							{
								Created:   wrapTimePoint(time.Now()),
								CreatedBy: "/bin/sh",
								Comment:   "FROM cloud-image-registry.cn-shanghai.cr.aliyuncs.com/applications/1137465199671599:cnstack-acos-0.0.5-beta-07b8866-70f202170cbcb3",
							},
						},
					},
					ImageExtension: ImageExtension{
						BuildClient: BuildClient{
							SealerVersion:  "0.9",
							BuildahVersion: "0.0.1",
						},
						SchemaVersion: "0.1",
						Type:          "app-image",
						Applications: []version.VersionedApplication{
							&v1.Application{
								NameVar: "app1",
								TypeVar: application.ShellApp,
								FilesVar: []string{
									"cnstack-acos-0.0.5-beta-install.sh",
								},
								VersionVar: "v1",
							},
							&v1.Application{
								NameVar: "app2",
								TypeVar: application.HelmApp,
								FilesVar: []string{
									"chart.tgz",
								},
								VersionVar: "v1",
							},
							&v1.Application{
								NameVar: "app3",
								TypeVar: application.KubeApp,
								FilesVar: []string{
									"mysql.yaml",
								},
								VersionVar: "v1",
							},
						},
						Launch: Launch{
							Cmds: nil,
							AppNames: []string{
								"app1",
								"app2",
							},
						},
					},
				},
			},
			want: `{"id":"783d1c3814b9cb28dafd7c3ca34b734d5798a61dc523c0d30349636e7cd56cc6","name":"cloud-image-registry.cn-shanghai.cr.aliyuncs.com/applications/1137465199671599:cnstack-acos-0.0.5-beta-07b8866-70f202","digest":"sha256:ebdd1badbb67f0f3de27317252618759db11782928bd206e9b155d5cc81f2076","manifestv1":{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:783d1c3814b9cb28dafd7c3ca34b734d5798a61dc523c0d30349636e7cd56cc6","size":1288,"platform":{"architecture":"","os":""}},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","digest":"sha256:4e36d2017e0a3b3e6642471bbcefc8290fc92a173bd3a5e70d9285fe4af6cb51","size":3077,"platform":{"architecture":"","os":""}},{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","digest":"sha256:4086b218e01c8713ed47e93f8fd24c2e714a609ff3b5268c91231dd5590e58e4","size":288,"platform":{"architecture":"","os":""}}],"annotations":{"org.opencontainers.image.base.digest":"sha256:d3801a216e40f5bc517b7e8aa1da41c31127f819f38a49356ffbd23a3d476017","sealer.image.extension":"{\\\"type\\\":\\\"app-installer\\\",\\\"applications\\\":[{\\\"name\\\":\\\"cnstack-acos-0.0.5-beta\\\",\\\"type\\\":\\\"shell\\\",\\\"launchfiles\\\":[\\\"cnstack-acos-0.0.5-beta-install.sh\\\"],\\\"version\\\":\\\"v1\\\"}],\\\"launch\\\":{\\\"cmds\\\":[\\\"bash application/apps/cnstack-acos-0.0.5-beta/cnstack-acos-0.0.5-beta-install.sh\\\"]}}"}},"ociv1":{"created":"2022-12-14T13:02:50.93707+08:00","architecture":"amd64","os":"linux","config":{"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Labels":{"io.buildah.version":"1.27.1","io.sealer.version":"0.9"}},"rootfs":{"type":"layers","diff_ids":["sha256:aa735d9c504ee5a122bb323c0147a1ad317bd0e72b5eef1094025e793f6f4912","sha256:7c5cf4cbab5b83e603116d3b3f84287df4f8e70c3c35e3cd6d384b451bb8c26d","sha256:e0af0977fcb45ba29be4ef74ec1dcdbb625e01b78905bb089b8cc6b0ffc717ba"]},"history":[{"created":"2022-12-14T13:02:50.93707+08:00","created_by":"/bin/sh -c #(nop) COPY dir:2127a0f14734aff8fb3128a35824dcbb01dd2d0f8295e55e3ea0ab299daef971 in applications "},{"created":"2022-12-14T13:02:50.93707+08:00","created_by":"/bin/sh -c #(nop) COPY file:b48c5033638953a52b8c36a41734d2e862787520a3b372e4ef1161d491c7221f in application/apps/cnstack-acos-0.0.5-beta/ ","comment":"FROM c6dfe3236bb0"},{"created":"2022-12-14T13:02:50.93707+08:00","created_by":"/bin/sh -c #(nop) LABEL \"io.sealer.version\"=\"0.9\"","comment":"FROM 4a18afda8bf5","empty_layer":true},{"created":"2022-12-14T13:02:50.937071+08:00","created_by":"/bin/sh","comment":"FROM cloud-image-registry.cn-shanghai.cr.aliyuncs.com/applications/1137465199671599:cnstack-acos-0.0.5-beta-07b8866-70f202170cbcb3"}]},"buildClient":{"sealerVersion":"0.9","buildahVersion":"0.0.1"},"schemaVersion":"0.1","type":"app-image","applications":[{"name":"app1","type":"shell","launchfiles":["cnstack-acos-0.0.5-beta-install.sh"],"version":"v1"},{"name":"app2","type":"helm","launchfiles":["chart.tgz"],"version":"v1"},{"name":"app3","type":"kube","launchfiles":["mysql.yaml"],"version":"v1"}],"launch":{"app_names":["app1","app2"]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.args.spec)
			if err != nil {
				t.Errorf("failed to marshal image spec: %v", err)
				return
			}
			result := string(got)
			if reflect.DeepEqual(result, tt.want) {
				t.Errorf("NewImageSpec() = %v, want %v", result, tt.want)
			}
		})
	}
}

func wrapTimePoint(t time.Time) *time.Time {
	return &t
}
