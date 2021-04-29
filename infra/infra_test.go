package infra

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
	"io/ioutil"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestApply(t *testing.T) {

	cluster := v1.Cluster{}

	yamlFile, err := ioutil.ReadFile("./Clusterfile")
	if err != nil {
		t.Errorf("read yaml file get an err #%v", err)

	}
	err = yaml.Unmarshal(yamlFile, &cluster)

	if err != nil {
		t.Errorf("read yaml file get an err #%v", err)

	}

	aliProvider := NewDefaultProvider(&cluster)
	err = aliProvider.Apply(&cluster)
	if err != nil {
		fmt.Printf("%v", err)
	}
	data, err := yaml.Marshal(&cluster)
	err = ioutil.WriteFile("./Clusterfile", data, 0777)
}

func TestGetAKSKFromEnv(t *testing.T) {
	config := Config{}
	GetAKSKFromEnv(&config)
	fmt.Printf("%v", config)

}

func TestDeleteInstances(t *testing.T) {
	config := Config{}
	GetAKSKFromEnv(&config)
	client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)

	request := ecs.CreateDeleteInstancesRequest()
	request.Scheme = "https"
	request.Force = requests.NewBoolean(true)
	request.InstanceId = &[]string{
		"i-uf69n04nc0osynamesn2",
		"i-uf6hdybz4rotahiwj8kn",
		"i-uf6a0srss899w5yn2h6t",
		"i-uf6a0srss899w5yn2h6u",
		"i-uf6a0srss899w5yn2h6v",
		"i-uf6a0srss899w5yn2h6s",
		"i-uf6gnqwhe5mez7qelg3l",
		"i-uf6gnqwhe5mez7qelg3m",
		"i-uf6gnqwhe5mez7qelg3k",
		"i-uf6gnqwhe5mez7qelg3n",
		"i-uf6gn4h06jw37ifq9ki6",
		"i-uf68m7sc6s68nm383gyi",
	}
	response, err := client.DeleteInstances(request)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Printf("response is %#v\n", response)
}
func TestDeleteSecurityGroup(t *testing.T) {
	config := Config{}
	GetAKSKFromEnv(&config)
	securityGroupIds := []string{
		"sg-uf69n04nc0osyp9mial3",
		"sg-uf6c37xtwtghzabjfh0k",
		"sg-uf6duslviuhlvbyr74ke",
		"sg-uf68jz5neq6tx2rf1mqu",
		"sg-uf6c23yx9aqoxn8eemxb",
	}
	for _, id := range securityGroupIds {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)

		request := ecs.CreateDeleteSecurityGroupRequest()
		request.Scheme = "https"

		request.SecurityGroupId = id

		response, err := client.DeleteSecurityGroup(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}

}

func TestDeleteVswitch(t *testing.T) {
	config := Config{}
	GetAKSKFromEnv(&config)
	vSwitchIDs := []string{
		"vsw-uf68dq62th7irzg48kb5a",
		"vsw-uf651u6e8lg5wfw94t930",
		"vsw-uf6s50sqpnbx30o2hu88s",
		"vsw-uf6wnyt5anmw4fcn5zss1",
		"vsw-uf6g7cmjbwshuxwkjz0wa",
	}
	for _, vSwitchID := range vSwitchIDs {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)

		request := ecs.CreateDeleteVSwitchRequest()
		request.Scheme = "https"

		request.VSwitchId = vSwitchID

		response, err := client.DeleteVSwitch(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}

}

func TestDeleteVpc(t *testing.T) {

	config := Config{}
	GetAKSKFromEnv(&config)
	vpcids := []string{
		"vpc-uf6gvyids3riounf07d3p",
		"vpc-uf612vsqrwspobp0kjs5t",
		"vpc-uf6890nbfhcxrxtntombm",
		"vpc-uf68ybg8gjvp8t0vkfsbs",
		"vpc-uf610t8krl6dhc3ghtdma",
	}
	for _, vpcid := range vpcids {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)

		request := ecs.CreateDeleteVpcRequest()
		request.Scheme = "https"

		request.VpcId = vpcid

		response, err := client.DeleteVpc(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}

}

func TestGetEIP(t *testing.T) {
	config := Config{}
	err := GetAKSKFromEnv(&config)
	client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)

	request := ecs.CreateAllocateEipAddressRequest()
	request.Scheme = "https"

	response, err := client.AllocateEipAddress(request)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Printf("response is %#v\n", response)
}

func TestReleaseEIP(t *testing.T) {
	config := Config{}
	GetAKSKFromEnv(&config)
	client, _ := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
	eipid := []string{
		"eip-uf66uj4susq4lfmcyym8n",
		"eip-uf6nvcr1zomdj805digd3",
		"eip-uf6gpfd9h0bji4oj6qfqi",
		"eip-uf6fh5mtal0mzai3higj3",
	}
	for _, s := range eipid {
		request := ecs.CreateReleaseEipAddressRequest()
		request.Scheme = "https"

		request.AllocationId = s

		response, err := client.ReleaseEipAddress(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}

}

func TestSort(t *testing.T) {
	iplist := []string{"192.168.0.3", "192.168.0.16", "192.168.0.4", "192.168.0.1"}
	utils.SortIPList(iplist)
	fmt.Printf("%v", iplist)
}
