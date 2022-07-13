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

//Metadata use file Metadata in rootfs to help cluster install.
type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	//KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion"`
	NydusFlag   bool   `json:"NydusFlag"`
	//ClusterRuntime is a flag to distinguish the runtime for k0s、k8s、k3s
	ClusterRuntime ClusterRuntime `json:"ClusterRuntime"`
}

type ClusterRuntime string

const (
	K0s ClusterRuntime = "k0s"
	K3s ClusterRuntime = "k3s"
	K8s ClusterRuntime = "k8s"
)
