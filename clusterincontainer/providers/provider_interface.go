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

@changelog The Sealer Authors
1. change the parameters of Provision method of Provider interface
2. delete the Info method from Provider interface
*/

package providers

import "github.com/alibaba/sealer/clusterincontainer/types"

// Provider represents a provider of cluster / node infrastructure
type Provider interface {
	// Provision should create and start the nodes, just short of
	// actually starting up Kubernetes, based on the given cluster config
	Provision(cluster types.Cluster) error
	// ListClusters discovers the clusters that currently have resources
	// under this providers
	ListClusters() ([]string, error)
	// ListNodes returns the nodes under this provider for the given
	// cluster name, they may or may not be running correctly
	ListNodes(cluster string) ([]types.Node, error)
	// DeleteNodes deletes the provided list of nodes
	// These should be from results previously returned by this provider
	// E.G. by ListNodes()
	DeleteNodes([]types.Node) error
	// GetAPIServerEndpoint returns the host endpoint for the cluster's API server
	GetAPIServerEndpoint(cluster string) (string, error)
	// CollectLogs will populate dir with cluster logs and other debug files
	CollectLogs(dir string, nodes []types.Node) error
}
