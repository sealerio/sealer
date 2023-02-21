// Copyright © 2023 Alibaba Group Holding Ltd.
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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/sealerio/sealer/build/kubefile/command"

	image_v1 "github.com/sealerio/sealer/pkg/define/image/v1"
)

func TestGetImageExtensionFromAnnotations(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    image_v1.ImageExtension
		wantErr bool
	}{
		{
			name: "",
			args: args{
				annotations: map[string]string{
					"sealer.image.extension": `{"buildClient":{"sealerVersion":"0.9.0","buildahVersion":"1.27.1"},"schemaVersion":"v1alpha1","type":"kube-installer","launch":{"cmds":["ls","-l"]}}`,
				},
			},
			want: image_v1.ImageExtension{
				BuildClient: image_v1.BuildClient{
					SealerVersion:  "0.9.0",
					BuildahVersion: "1.27.1",
				},
				SchemaVersion: "v1alpha1",
				Type:          "kube-installer",
				Applications:  nil,
				Launch: image_v1.Launch{
					Cmds:     []string{"ls", "-l"},
					AppNames: nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImageExtensionFromAnnotations(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetImageExtensionFromAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if !reflect.DeepEqual(gotJSON, wantJSON) {
				t.Errorf("GetImageExtensionFromAnnotations() got = %s\n, want %s", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestGetContainerImagesFromAnnotations(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []*image_v1.ContainerImage
		wantErr bool
	}{
		{
			name: "",
			args: args{
				annotations: map[string]string{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				annotations: map[string]string{
					image_v1.SealerImageContainerImageList: ``,
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				annotations: map[string]string{
					image_v1.SealerImageContainerImageList: `[{"image":"nginx:latest","appName":"nginx"},{"image":"registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest","appName":"dashboard"},{"image":"busybox:latest"}]`,
				},
			},
			want: []*image_v1.ContainerImage{
				{
					Image:   "nginx:latest",
					AppName: "nginx",
				},
				{
					Image:   "registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest",
					AppName: "dashboard",
				},
				{
					Image: "busybox:latest",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getContainerImagesFromAnnotations(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContainerImagesFromAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetContainerImagesFromAnnotations() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleLabelOutput(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "",
			args: args{
				labels: map[string]string{
					fmt.Sprintf("%s%s", command.LabelKubeCNIPrefix, "calico"):      "true",
					fmt.Sprintf("%s%s", command.LabelKubeCNIPrefix, "flannel"):     "true",
					fmt.Sprintf("%s%s", command.LabelKubeCSIPrefix, "alibaba-csi"): "true",
					"key1": "value1",
				},
			},
			want: map[string]string{
				command.LabelSupportedKubeCNIAlpha: `["calico","flannel"]`,
				command.LabelSupportedKubeCSIAlpha: `["alibaba-csi"]`,
				"key1":                             "value1",
			},
		},
		{
			name: "",
			args: args{
				labels: map[string]string{},
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleImageLabelOutput(tt.args.labels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleLabelOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}
