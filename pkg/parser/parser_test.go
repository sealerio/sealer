// Copyright © 2021 Alibaba Group Holding Ltd.
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

package parser

import (
	"reflect"
	"testing"

	"github.com/alibaba/sealer/version"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func Test_decodeLine(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"test FROM command",
			args{line: "FROM kubernetes:1.18.2"},
			"FROM",
			"kubernetes:1.18.2",
			false,
		},
		{
			"test FROM command",
			args{line: " FROM kubernetes:1.18.2"},
			"FROM",
			"kubernetes:1.18.2",
			false,
		},
		{
			"test FROM command",
			args{line: "FROM kubernetes:1.18.2 "},
			"FROM",
			"kubernetes:1.18.2",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := decodeLine(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeLine() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("decodeLine() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	kubeFile := []byte(`FROM kubernetes:1.18.1

# this is annotations
COPY dashboard .
RUN echo "Config ssh ..." \
    && echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
RUN kubectl apply -f dashboard`)

	type args struct {
		kubeFile []byte
		name     string
	}
	tests := []struct {
		name string
		args args
		want *v1.Image
	}{
		{
			"test kubeFile",
			args{
				kubeFile: kubeFile,
				name:     "",
			},
			&v1.Image{
				TypeMeta: metaV1.TypeMeta{APIVersion: "", Kind: "Image"},
				Spec: v1.ImageSpec{
					SealerVersion: version.Get().GitVersion,
					Layers: []v1.Layer{
						{
							Type:  "FROM",
							Value: "kubernetes:1.18.1",
						},
						{
							Type:  "COPY",
							Value: "dashboard .",
						},
						{
							Type:  "RUN",
							Value: "echo \"Config ssh ...\"     && echo \"PermitRootLogin yes\" >> /etc/ssh/sshd_config",
						},
						{
							Type:  "RUN",
							Value: "kubectl apply -f dashboard",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{}
			got, err := p.Parse(tt.args.kubeFile)
			if err != nil {
				t.Errorf("Parse() error = %v ", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
