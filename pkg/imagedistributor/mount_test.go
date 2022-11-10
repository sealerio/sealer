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

package imagedistributor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageMountDir(t *testing.T) {
	tests := []struct {
		testName   string
		imageName  string
		wantedPath string
	}{
		{testName: "image name with tag", imageName: "nginx:v1", wantedPath: "/var/lib/sealer/data/mount/nginx:v1"},
		{testName: "image name with repo and tag", imageName: "library/nginx:v1", wantedPath: "/var/lib/sealer/data/mount/library_nginx:v1"},
		{testName: "image name with domain repo and tag", imageName: "docker.io/library/nginx:v1", wantedPath: "/var/lib/sealer/data/mount/docker.io_library_nginx:v1"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mountedPath := imageMountDir(tt.imageName)
			assert.Equal(t, tt.wantedPath, mountedPath)
		})
	}
}
