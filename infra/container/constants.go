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

package container

const (
	DOCKER    = "docker"
	CONTAINER = "CONTAINER"
)
const (
	NETWROKID           = "NetworkId"
	IMAGEID             = "ImageId"
	DefaultPassword     = "Seadent123"
	ResourceNetwork     = "network"
	ResourceImage       = "image"
	DefaultNetworkName  = "sealer-network"
	DefaultImageName    = "registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image:latest"
	DockerHost          = "/var/run/docker.sock"
	MASTER              = "master"
	NODE                = "node"
	SealerImageRootPath = "/var/lib/sealer"
	ChangePasswordCmd   = "echo root:%s | chpasswd" // #nosec
	RoleLabel           = "sealer-io-role"
	RoleLabelMaster     = "sealer-io-role-is-master"
)
