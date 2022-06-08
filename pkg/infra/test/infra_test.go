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

/*func TestApply(t *testing.T) {
	//setup cluster
	password := os.Getenv("SealerPassword")
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
				Passwd: password,
			},
		},
	}

	aliProvider, err := infra.NewDefaultProvider(&cluster)
	if err != nil {
		fmt.Printf("%v", err)
	} else {
		fmt.Printf("%v", aliProvider.Apply())
	}

	t.Run("modify instance type", func(t *testing.T) {
		cluster.Spec.Masters.CPU = "4"
		cluster.Spec.Masters.Memory = "8"
		cluster.Spec.Nodes.CPU = "4"
		cluster.Spec.Nodes.Memory = "8"
		fmt.Printf("%v", aliProvider.Apply())
	})

	t.Run("add instance count", func(t *testing.T) {
		cluster.Spec.Masters.Count = "5"
		cluster.Spec.Nodes.Count = "5"
		fmt.Printf("%v", aliProvider.Apply())
		fmt.Printf("%v \n", cluster.Spec.Masters)
		fmt.Printf("%v \n", cluster.Spec.Nodes)
	})

	t.Run("reduce instance count", func(t *testing.T) {
		cluster.Spec.Masters.Count = "1"
		cluster.Spec.Nodes.Count = "1"
		fmt.Printf("%v", aliProvider.Apply())
	})

	t.Run("modify instance type & count both", func(t *testing.T) {
		cluster.Spec.Masters.CPU = "8"
		cluster.Spec.Masters.Memory = "16"
		cluster.Spec.Nodes.CPU = "8"
		cluster.Spec.Nodes.Memory = "16"
		cluster.Spec.Masters.Count = "5"
		cluster.Spec.Nodes.Count = "5"
		fmt.Printf("%v", aliProvider.Apply())
	})

	// todo
	t.Run("modify instance system disk", func(t *testing.T) {

	})

	//teardown
	time.Sleep(60 * time.Second)
	now := metav1.Now()
	cluster.ObjectMeta.DeletionTimestamp = &now
	fmt.Printf("%v", aliProvider.Apply())
}
*/
