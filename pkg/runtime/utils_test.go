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
	"gotest.tools/v3/assert"
	"os"
	"syscall"
	"testing"
)

func TestLoadMetadata(t *testing.T) {
	const (
		rootfsPath       = "/var/lib/sealer/my-cluster/rootfs"
		mockMeatadata    = "{\n  \"version\": \"v1.19.8\",\n  \"arch\": \"amd64\"\n}"
		metadataFileName = "Metadata"
	)

	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	rootfs, err := os.MkdirTemp(rootfsPath, metadataFileName)
	assert.NilError(t, err)
	defer os.RemoveAll(rootfs)

	if err = os.WriteFile(rootfs, []byte(mockMeatadata), 0666); err != nil {
		t.Errorf("write temp file in %s error: %s", rootfs, err)
	}

	metadata, err := LoadMetadata(rootfsPath)
	assert.Equal(t, "v1.19.8", metadata.KubeVersion)
	assert.Equal(t, "amd64", metadata.Arch)
}
