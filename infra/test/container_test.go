package infra

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/alibaba/sealer/infra/container"
	v1 "github.com/alibaba/sealer/types/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetUpClient() (*container.DockerProvider, error) {
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
				Passwd: "kakazhou719",
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

	t.Run("apply docker image", func(t *testing.T) {
		err := client.PrepareImageResource()
		if err != nil {
			t.Logf("apply docker images failed %v", err)
			return
		}
		imageId := client.GetImageIdByName(client.ImageResource.DefaultName)
		if imageId == "" {
			t.Logf("check failed. image:%s not found", client.ImageResource.DefaultName)
			return
		}

		resp, err := client.GetImageResourceById(imageId)
		if err != nil {
			t.Logf("get image info failed %v", err)
			return
		}
		t.Logf("get image info sucess %s %s", resp.ID, resp.RepoTags)

		err = client.DeleteImageResource(imageId)
		if err != nil {
			t.Logf("check failed.failed to delete image %s.error is %v", client.ImageResource.DefaultName, err)
			return
		}
		t.Logf("succuss to apply docker image")
	})

	t.Run("apply docker network", func(t *testing.T) {
		err := client.PrepareNetworkResource()
		if err != nil {
			t.Logf("apply docker network failed %v", err)
			return
		}

		net, err := client.GetNetworkResourceById(client.NetworkResource.Id)
		if err != nil {
			t.Logf("get network info failed %v", err)
			return
		}
		if net.Name != client.NetworkResource.DefaultName {
			t.Logf("check failed. network: %s not found", client.NetworkResource.DefaultName)
			return
		}

		t.Logf("get network info sucess %s %s", net.Name, net.ID)
		err = client.DeleteNetworkResource(client.NetworkResource.Id)
		if err != nil {
			t.Logf("check failed.failed to delete network %s.error is %v", client.NetworkResource.DefaultName, err)
			return
		}
		t.Logf("succuss to apply docker network")
	})

	t.Run("apply docker container", func(t *testing.T) {
		err := client.PrepareBaseResource()
		if err != nil {
			t.Logf("failed to prepare base resource %v", err)
			return
		}
		id, err := client.RunContainer(&container.CreateOptsForContainer{
			ContainerName:     "test-container",
			ContainerHostName: "test-container-host-name",
		})
		if err != nil {
			t.Logf("failed to run container %v", err)
			return
		}

		info, err := client.GetContainerInfo(id)
		if err != nil {
			t.Logf("failed to get container info of %s ,error is %v", id, err)
			return
		}
		if info.Status != "running" {
			t.Logf("failed to get container info %s,container is %v", id, info.Status)
			return
		}
		err = client.RmContainer(id)
		if err != nil {
			t.Logf("failed to delete container:%s", id)
			return
		}

		fmt.Println(client.Cluster.Annotations)
		deleteNetErr := client.DeleteNetworkResource(client.Cluster.Annotations[container.NETWROK_ID])
		deleteImageErr := client.DeleteImageResource(client.Cluster.Annotations[container.IMAGE_ID])

		if deleteNetErr != nil || deleteImageErr != nil {
			t.Logf("clean up err: %v %v", deleteNetErr, deleteImageErr)
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
				Passwd: "kakazhou719",
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
	// network id ,image id should not be nil
	// len(iplist) must equal count
	masterCount, err := strconv.Atoi(cluster.Spec.Masters.Count)
	if err != nil {
		return true
	}
	nodeCount, err := strconv.Atoi(cluster.Spec.Nodes.Count)
	if err != nil {
		return true
	}

	if cluster.Annotations[container.IMAGE_ID] == "" ||
		cluster.Annotations[container.NETWROK_ID] == "" ||
		len(cluster.Spec.Masters.IPList) != masterCount ||
		len(cluster.Spec.Nodes.IPList) != nodeCount {
		return true
	}

	return false
}
