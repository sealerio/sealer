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

package save

import (
	"context"
	"testing"
)

func TestSaveImages(t *testing.T) {
	images := []string{"ubuntu", "ubuntu:18.04", "registry.aliyuncs.com/google_containers/coredns:1.6.5", "fanux/lvscare"}
	is := NewImageSaver(context.Background())
	err := is.SaveImages(images, "/var/lib/registry", "amd64")
	if err != nil {
		t.Error(err)
	}
}

func Test_splitDockerDomain(t *testing.T) {
	names := []string{"docker.io/library/alpine:latest", "k8s.gcr.io/kube-apiserver", "ubuntu"}
	domain, remainder := splitDockerDomain(names[0])
	if domain != defaultDomain || remainder != "library/alpine:latest" {
		t.Errorf("split %s error", names[0])
	}
	domain, remainder = splitDockerDomain(names[1])
	if domain != "k8s.gcr.io" || remainder != "kube-apiserver" {
		t.Errorf("split %s error", names[1])
	}
	domain, remainder = splitDockerDomain(names[2])
	if domain != defaultDomain || remainder != "library/ubuntu" {
		t.Errorf("split %s error", names[2])
	}
}

func Test_parseNormalizedNamed(t *testing.T) {
	names := []string{"docker.io/library/alpine:latest", "k8s.gcr.io/kube-apiserver", "ubuntu"}
	named, err := parseNormalizedNamed(names[0])
	if err != nil || named.domain != defaultDomain || named.repo != "library/alpine" || named.tag != defaultTag {
		t.Errorf("parse %s error", names[0])
	}
	named, err = parseNormalizedNamed(names[1])
	if err != nil || named.domain != "k8s.gcr.io" || named.repo != "kube-apiserver" || named.tag != defaultTag {
		t.Errorf("parse %s error", names[1])
	}
	named, err = parseNormalizedNamed(names[2])
	if err != nil || named.domain != defaultDomain || named.repo != "library/ubuntu" || named.tag != defaultTag {
		t.Errorf("parse %s error", names[2])
	}
}
