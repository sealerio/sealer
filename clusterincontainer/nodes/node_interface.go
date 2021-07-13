/*
Copyright © 2021 Alibaba Group Holding Ltd.
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodes

import (
	"io"

	"github.com/alibaba/sealer/clusterincontainer/types"
)

// Node represents a cluster node which is a container running node image
type Node interface {
	// The node should implement types.Cmder for running commands against the node
	types.Cmder
	// String should return the node name
	String() string
	// Role should return the node's role
	Role() (string, error)
	IP() (ipv4 string, ipv6 string, err error)
	// SerialLogs collects the "node" container logs
	SerialLogs(writer io.Writer) error
}
