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

package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sealerio/sealer/common"
)

const (
	mockMetadata = `{
  "version": "v1.19.8",
  "arch": "amd64",
  "ClusterRuntime": "kubernetes",
  "NydusFlag": false,
  "kubeVersion": "",
  "variant": ""
}
`
)

func TestLoadMetadata(t *testing.T) {
	const (
		rootfsPath       = "rootfs"
		metadataFileName = "Metadata"
	)
	type object struct {
		RuntimeMetadata []byte
	}
	tests := []struct {
		name    string
		object  object
		want    *Metadata
		wantErr bool
	}{
		{
			name: "test metadata file from rootfs",
			object: object{
				[]byte(mockMetadata),
			},
			want: &Metadata{
				Version:        "v1.19.8",
				Arch:           "amd64",
				ClusterRuntime: ClusterRuntime(common.K8s),
				NydusFlag:      false,
				KubeVersion:    "",
				Variant:        "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "test-rootfs-metadata-tmp")
			if err != nil {
				t.Errorf("Make temp dir %s error = %s, wantErr %v", dir, err, tt.wantErr)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Errorf("Remove temp dir %s error = %v, wantErr %v", dir, err, tt.wantErr)
				}
			}()

			err = os.Mkdir(filepath.Join(dir, rootfsPath), 0777)
			if err != nil {
				t.Errorf("Make dir %s error = %s, wantErr %v", dir, err, tt.wantErr)
			}

			err = os.WriteFile(filepath.Join(dir, rootfsPath, metadataFileName), tt.object.RuntimeMetadata, 0777)
			if err != nil {
				t.Errorf("Write file %s error = %v, wantErr %v", filepath.Join(dir, rootfsPath, metadataFileName), err, tt.wantErr)
			}

			metadata, err := LoadMetadata(filepath.Join(dir, rootfsPath))
			if err != nil {
				t.Errorf("LoadMetadata error: %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(metadata, tt.want) {
				t.Errorf("Metadata loaded from file is not wanted! Got: %v, wanted: %v", metadata, tt.want)
			}
		})
	}
}
