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

package runtime

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sealerio/sealer/common"
	osi "github.com/sealerio/sealer/utils/os"
)

// Deprecated
type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion"`
	NydusFlag   bool   `json:"NydusFlag"`
	// ClusterRuntime is a flag to distinguish the runtime for k0s、k8s、k3s
	ClusterRuntime ClusterRuntime `json:"ClusterRuntime"`
}

type ClusterRuntime string

// Deprecated
func LoadMetadata(rootfs string) (*Metadata, error) {
	metadataPath := filepath.Join(rootfs, common.DefaultMetadataName)
	var metadataFile []byte
	var err error
	var md Metadata
	if !osi.IsFileExist(metadataPath) {
		return nil, nil
	}

	metadataFile, err = os.ReadFile(filepath.Clean(metadataPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read ClusterImage metadata: %v", err)
	}
	err = json.Unmarshal(metadataFile, &md)
	if err != nil {
		return nil, fmt.Errorf("failed to load ClusterImage metadata: %v", err)
	}
	return &md, nil
}

// Deprecated
func GetClusterImagePlatform(rootfs string) (cp ocispecs.Platform) {
	// current we only support build on linux
	cp = ocispecs.Platform{
		Architecture: "amd64",
		OS:           "linux",
		Variant:      "",
		OSVersion:    "",
	}
	meta, err := LoadMetadata(rootfs)
	if err != nil {
		return
	}
	if meta == nil {
		return
	}
	if meta.Arch != "" {
		cp.Architecture = meta.Arch
	}
	if meta.Variant != "" {
		cp.Variant = meta.Variant
	}
	return
}

func RemoteCertCmd(altNames []string, hostIP net.IP, hostName, serviceCIRD, DNSDomain string) string {
	cmd := "seautil cert gen "
	if hostIP != nil {
		cmd += fmt.Sprintf(" --node-ip %s", hostIP.String())
	}

	if hostName != "" {
		cmd += fmt.Sprintf(" --node-name %s", hostName)
	}

	if serviceCIRD != "" {
		cmd += fmt.Sprintf(" --service-cidr %s", serviceCIRD)
	}

	if DNSDomain != "" {
		cmd += fmt.Sprintf(" --dns-domain %s", DNSDomain)
	}

	for _, name := range append(altNames, common.APIServerDomain) {
		if name != "" {
			cmd += fmt.Sprintf(" --alt-names %s", name)
		}
	}

	return cmd
}

func IsInContainer() bool {
	data, err := osi.NewFileReader("/proc/1/environ").ReadAll()
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "container=docker")
}
