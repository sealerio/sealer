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

package main

import (
	"fmt"
	"github.com/alibaba/sealer/client/k8s"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type nodes string

func (n *nodes) Exec(cluster *v1.Cluster) error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	nodeList, err := client.ListNodes()
	if err != nil {
		return fmt.Errorf("cluster nodes not found, %v", err)
	}
	for _, v := range nodeList.Items {
		fmt.Println(v.Name)
	}
	return nil
}

// Plugin is the exposed variable sealer will look up it.
var Plugin nodes
