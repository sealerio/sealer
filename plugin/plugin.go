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

package plugin

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Interface interface {
	Run(context Context, phase Phase) error
	GetPluginType() string
}

type Phase string

const (
	PhasePreInit     = Phase("PreInit")
	PhasePreInstall  = Phase("PreInstall")
	PhasePostInstall = Phase("PostInstall")
	PhaseOriginally  = Phase("Originally")
	PhasePreGuest    = Phase("PreGuest")
)

const (
	// Plugin used for golang so file to find the related symbol
	Plugin             = "Plugin"
	EtcdPlugin         = "ETCD"
	LabelPlugin        = "LABEL"
	ShellPlugin        = "SHELL"
	HostNamePlugin     = "HOSTNAME"
	ClusterCheckPlugin = "CLUSTERCHECK"
)

const (
	ClusterNotReady = "ClusterNotReady"
	ClusterReady    = "ClusterReady"
)

type Context struct {
	Cluster *v1.Cluster
	Plugin  *v1.Plugin
}
