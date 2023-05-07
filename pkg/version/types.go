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

package version

import (
	"fmt"
)

// Info contains versioning information.
// TODO: Add []string of api versions supported? It's still unclear
// how we'll want to distribute that information.
type Info struct {
	Major        string `json:"major,omitempty"`
	Minor        string `json:"minor,omitempty"`
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit,omitempty"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

type Output struct {
	SealerVersion     Info               `json:"sealerVersion,omitempty" yaml:"sealerVersion,omitempty"`
	CriRuntimeVersion *CriRuntimeVersion `json:"criVersionInfo,omitempty" yaml:"criVersionInfo,omitempty"`
	KubernetesVersion *KubernetesVersion `json:"kubernetesVersionInfo,omitempty" yaml:"kubernetesVersionInfo,omitempty"`
	K0sVersion        *k0sVersion        `json:"k0sVersionInfo,omitempty" yaml:"k0sVersionInfo,omitempty"`
	K3sVersion        *k3sVersion        `json:"k3sVersionInfo,omitempty" yaml:"k3sVersionInfo,omitempty"`
}

type CriRuntimeVersion struct {
	// Version of the kubelet runtime API.
	Version string `json:"Version,omitempty" yaml:"Version,omitempty"`
	// Name of the container runtime.
	RuntimeName string `json:"RuntimeName,omitempty" yaml:"RuntimeName,omitempty"`
	// Version of the container runtime. The string must be
	// semver-compatible.
	RuntimeVersion string `json:"RuntimeVersion,omitempty" yaml:"RuntimeVersion,omitempty"`
	// API version of the container runtime. The string must be
	// semver-compatible.
	RuntimeAPIVersion string `json:"RuntimeApiVersion,omitempty" yaml:"RuntimeApiVersion,omitempty"`
}

type KubernetesVersion struct {
	ClientVersion    *KubectlInfo `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	KustomizeVersion string       `json:"kustomizeVersion,omitempty" yaml:"kustomizeVersion,omitempty"`
	ServerVersion    *KubectlInfo `json:"serverVersion,omitempty" yaml:"serverVersion,omitempty"`
}

type k0sVersion struct {
	K0sVersion string `json:"k9sVersion,omitempty" yaml:"k0sVersion,omitempty"`
}

type k3sVersion struct {
	K3sVersion string `json:"k3sVersion,omitempty" yaml:"k3sVersion,omitempty"`
}

type KubectlInfo struct {
	Major        string `json:"major" yaml:"major"`
	Minor        string `json:"minor" yaml:"minor"`
	GitVersion   string `json:"gitVersion" yaml:"gitVersion"`
	GitCommit    string `json:"gitCommit" yaml:"gitCommit"`
	GitTreeState string `json:"gitTreeState" yaml:"gitTreeState"`
	BuildDate    string `json:"buildDate" yaml:"buildDate"`
	GoVersion    string `json:"goVersion" yaml:"goVersion"`
	Compiler     string `json:"compiler" yaml:"compiler"`
	Platform     string `json:"platform" yaml:"platform"`
}

// String returns info as a human-friendly version string.
func (info Info) String() string {
	if s, err := info.Text(); err == nil {
		return string(s)
	}

	return fmt.Sprintf("%s-%s", info.GitVersion, info.GitCommit)
}
