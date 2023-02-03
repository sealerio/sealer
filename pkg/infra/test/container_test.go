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

package infra

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sealerio/sealer/pkg/infra/container"
	dc "github.com/sealerio/sealer/pkg/infra/container/client"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

func SetUpClient() (*container.ApplyProvider, error) {
	cluster := &v1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "zlink.aliyun.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cluster",
		},
		Spec: v1.ClusterSpec{
			Masters: v1.Hosts{
				Count:      "1",
				CPU:        "2",
				Memory:     "4",
				SystemDisk: "100",
				DataDisks:  []string{"100"},
			},
			Nodes: v1.Hosts{
				Count:      "1",
				CPU:        "2",
				Memory:     "4",
				SystemDisk: "100",
				DataDisks:  []string{"100"},
			},
			Provider: container.CONTAINER,
			SSH: v1.SSH{
				Passwd: "zhy76",
			},
		},
	}

	client, err := container.NewClientWithCluster(cluster)

	if err != nil {
		fmt.Printf("new docker client failed,%v", err)
	}
	return client, nil
}

func TestContainerResource(t *testing.T) {
	//setup Container client
	client, _ := SetUpClient()
	t.Run("apply docker container", func(t *testing.T) {
		id, err := client.Provider.RunContainer(&dc.CreateOptsForContainer{
			ContainerName:     "test-container",
			ContainerHostName: "test-container-host-name",
			ImageName:         container.ImageName,
			NetworkName:       container.NetworkName,
		})
		if err != nil {
			t.Logf("failed to run container %v", err)
			return
		}

		info, err := client.Provider.GetContainerInfo(id, container.NetworkName)
		if err != nil {
			t.Logf("failed to get container info of %s ,error is %v", id, err)
			return
		}
		if info.Status != "running" {
			t.Logf("failed to get container info %s,container is %v", id, info.Status)
			return
		}
		err = client.Provider.RmContainer(id)
		if err != nil {
			t.Logf("failed to delete container:%s", id)
			return
		}

		t.Logf("succuss to apply docker container")
	})
}

func TestContainerApply(t *testing.T) {
	cluster := &v1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "zlink.aliyun.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cluster",
		},
		Spec: v1.ClusterSpec{
			Masters: v1.Hosts{
				Count:      "1",
				CPU:        "2",
				Memory:     "2",
				SystemDisk: "100",
				DataDisks:  []string{"100"},
			},
			Nodes: v1.Hosts{
				Count:      "1",
				CPU:        "2",
				Memory:     "2",
				SystemDisk: "100",
				DataDisks:  []string{"100"},
			},
			Provider: container.CONTAINER,
			SSH: v1.SSH{
				Passwd: "zhy76",
			},
		},
	}

	client, err := container.NewClientWithCluster(cluster)
	if err != nil {
		t.Logf("failed to new container client")
		return
	}
	// new apply: 1 master + 1 node
	err = client.Apply()
	if err != nil {
		t.Logf("failed to apply container infra %v", err)
		return
	}
	if CheckContainerApplyResult(cluster) {
		t.Logf("container infra does not meet expectation %+v", cluster)
		return
	}
	t.Logf("succuss to apply container infra")
	// change apply :scale up ,3 master + 3 node
	cluster.Spec.Masters.Count = "3"
	cluster.Spec.Nodes.Count = "3"
	err = client.Apply()
	if err != nil {
		t.Logf("failed to scale up container infra %v", err)
		return
	}
	if CheckContainerApplyResult(cluster) {
		t.Logf("container infra does not meet expectation %+v", cluster)
		return
	}
	t.Logf("succuss to scale up container infra")
	// change apply:scale down, 3 master + 1 node
	cluster.Spec.Masters.Count = "3"
	cluster.Spec.Nodes.Count = "1"
	err = client.Apply()
	if err != nil {
		t.Logf("failed to scale down container infra %v", err)
		return
	}
	if CheckContainerApplyResult(cluster) {
		t.Logf("container infra does not meet expectation %+v", cluster)
		return
	}
	t.Logf("succuss to scale down container infra")
	// delete apply
	time.Sleep(60 * time.Second)
	now := metav1.Now()
	cluster.ObjectMeta.DeletionTimestamp = &now
	fmt.Printf("%v", client.Apply())
	t.Logf("succuss to clean up container infra")
}

func CheckContainerApplyResult(cluster *v1.Cluster) bool {
	// return false if result do not meet expectation
	// len(iplist) must equal count
	masterCount, err := strconv.Atoi(cluster.Spec.Masters.Count)
	if err != nil {
		return true
	}
	nodeCount, err := strconv.Atoi(cluster.Spec.Nodes.Count)
	if err != nil {
		return true
	}

	if len(cluster.Spec.Masters.IPList) != masterCount ||
		len(cluster.Spec.Nodes.IPList) != nodeCount {
		return true
	}

	return false
}
