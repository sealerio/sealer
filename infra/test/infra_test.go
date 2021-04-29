package infra

import (
	"fmt"
	"gitlab.alibaba-inc.com/seadent/pkg/infra"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
	"time"
)

func TestApply(t *testing.T) {
	//setup cluster
	passwod := os.Getenv("SealerPassword")
	cluster := v1.Cluster{
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
			Provider: "ALI_CLOUD",
			SSH: v1.SSH{
				Passwd: passwod,
			},
		},
	}

	aliProvider := infra.NewDefaultProvider(&cluster)
	fmt.Printf("%v", aliProvider.Apply(&cluster))

	t.Run("modity instance type", func(t *testing.T) {
		cluster.Spec.Masters.CPU = "4"
		cluster.Spec.Masters.Memory = "8"
		cluster.Spec.Nodes.CPU = "4"
		cluster.Spec.Nodes.Memory = "8"
		fmt.Printf("%v", aliProvider.Apply(&cluster))
	})

	t.Run("add instance count", func(t *testing.T) {
		cluster.Spec.Masters.Count = "5"
		cluster.Spec.Nodes.Count = "5"
		fmt.Printf("%v", aliProvider.Apply(&cluster))
		fmt.Printf("%v \n", cluster.Spec.Masters)
		fmt.Printf("%v \n", cluster.Spec.Nodes)
	})

	t.Run("reduce instance count", func(t *testing.T) {
		cluster.Spec.Masters.Count = "1"
		cluster.Spec.Nodes.Count = "1"
		fmt.Printf("%v", aliProvider.Apply(&cluster))
	})

	t.Run("modify instance type & count both", func(t *testing.T) {
		cluster.Spec.Masters.CPU = "8"
		cluster.Spec.Masters.Memory = "16"
		cluster.Spec.Nodes.CPU = "8"
		cluster.Spec.Nodes.Memory = "16"
		cluster.Spec.Masters.Count = "5"
		cluster.Spec.Nodes.Count = "5"
		fmt.Printf("%v", aliProvider.Apply(&cluster))
	})

	// todo
	t.Run("modfiy instance systemdisk", func(t *testing.T) {

	})

	//teardown
	time.Sleep(60 * time.Second)
	now := metav1.Now()
	cluster.ObjectMeta.DeletionTimestamp = &now
	fmt.Printf("%v", aliProvider.Apply(&cluster))
}
