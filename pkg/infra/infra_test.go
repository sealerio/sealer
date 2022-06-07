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
	"testing"

	"github.com/sealerio/sealer/utils/net"
)

/*
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

	aliProvider, err := NewDefaultProvider(&cluster)
	if err != nil {
		fmt.Printf("%v", err)
	} else {
		err = aliProvider.Apply()
		if err != nil {
			fmt.Printf("%v", err)
		}
	}
	data, err := yaml.Marshal(&cluster)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	err = os.NewCommonWriter("./Clusterfile").WriteFile(data)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
}

func TestGetAKSKFromEnv(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	fmt.Printf("%v", config)
}

func TestDeleteInstances(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
	if client == nil {
		fmt.Printf("%v \n", err)
	}
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	request := ecs.CreateDeleteInstancesRequest()
	request.Scheme = aliyun.Scheme
	request.Force = requests.NewBoolean(true)
	request.InstanceId = &[]string{}
	response, err := client.DeleteInstances(request)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Printf("response is %#v\n", response)
}
func TestDeleteSecurityGroup(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	securityGroupIds := []string{
		"sg-hp38q702bczjtnb5qxdh",
		"sg-hp33dkye42vdg38i49mg",
		"sg-hp36xu038m1cqwcmltc7",
		"sg-hp38q702bczjt9hxyjjk",
		"sg-hp36xu038m1cqsekdpdy",
		"sg-hp3250tdy1vv64i866dv",
		"sg-hp36utl2950o9m7b0eg9",
		"sg-hp34sj0h93usb66rs5zq",
	}
	for _, id := range securityGroupIds {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
		if client == nil {
			fmt.Printf("%v \n", err)
		}
		if err != nil {
			fmt.Printf("%v \n", err)
		}
		request := ecs.CreateDeleteSecurityGroupRequest()
		request.Scheme = aliyun.Scheme
		request.SecurityGroupId = id
		response, err := client.DeleteSecurityGroup(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}
}

func TestDeleteVswitch(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	vSwitchIDs := []string{
		"vsw-hp3kk6dkf7msos0cdej2f",
		"vsw-hp31isullq0d3n32bd8bu",
		"vsw-hp3y1hzn0dxpiagf9pwc5",
		"vsw-hp37frkvd9hck3pyir2go",
		"vsw-hp33g2g8nhh9d72mx4w6o",
		"vsw-hp3mywuhbc77fpxagcft6",
		"vsw-hp3xfh2gv576nx26t59kn",
		"vsw-hp33c9qnqd73vehangsok",
		"vsw-hp38rwznx0y14xi48nu7y",
	}
	for _, vSwitchID := range vSwitchIDs {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
		if err != nil {
			fmt.Printf("%v \n", err)
		}
		if client == nil {
			fmt.Printf("%v \n", err)
		}
		request := ecs.CreateDeleteVSwitchRequest()
		request.Scheme = aliyun.Scheme
		request.VSwitchId = vSwitchID
		response, err := client.DeleteVSwitch(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}
}

func TestDeleteVpc(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	vpcids := []string{
		"vpc-hp3e6ckmb1ngp1hkob9cb",
		"vpc-hp3bw0rlgmuhdq68b1t6m",
		"vpc-hp3qh53u6w15psvh856xz",
		"vpc-hp3lii8nsnosi0bwt460o",
		"vpc-hp3rn2b8i05l4pt7ksmed",
		"vpc-hp35s2er96rn7lqiw9sgx",
		"vpc-hp3djex4ingvqv8um5jtv",
		"vpc-hp3byr74ugj7zg4r547ap",
		"vpc-hp33jer8cd3epf8mu5m0k",
		"vpc-hp35gdcac444eyrwbmv6z",
	}
	for _, vpcid := range vpcids {
		client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
		if client == nil {
			fmt.Printf("%v \n", err)
		}
		if err != nil {
			fmt.Printf("%v \n", err)
		}
		request := ecs.CreateDeleteVpcRequest()
		request.Scheme = aliyun.Scheme
		request.VpcId = vpcid
		response, err := client.DeleteVpc(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}
}

func TestGetEIP(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	client, err := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
	if client == nil {
		fmt.Printf("%v \n", err)
	}
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	request := ecs.CreateAllocateEipAddressRequest()
	request.Scheme = aliyun.Scheme
	response, err := client.AllocateEipAddress(request)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Printf("response is %#v\n", response)
}

func TestReleaseEIP(t *testing.T) {
	config := aliyun.Config{}
	err := aliyun.LoadConfig(&config)
	if err != nil {
		fmt.Printf("%v \n", err)
	}
	client, _ := ecs.NewClientWithAccessKey(config.RegionID, config.AccessKey, config.AccessSecret)
	eipid := []string{
		"eip-uf66uj4susq4lfmcyym8n",
		"eip-uf6nvcr1zomdj805digd3",
		"eip-uf6gpfd9h0bji4oj6qfqi",
		"eip-uf6fh5mtal0mzai3higj3",
	}
	for _, s := range eipid {
		request := ecs.CreateReleaseEipAddressRequest()
		request.Scheme = aliyun.Scheme
		request.AllocationId = s
		response, err := client.ReleaseEipAddress(request)
		if err != nil {
			fmt.Print(err.Error())
		}
		fmt.Printf("response is %#v\n", response)
	}
}
*/

func TestSort(t *testing.T) {
	iplist := []string{"192.168.0.3", "192.168.0.16", "192.168.0.4", "192.168.0.1"}
	net.SortIPList(iplist)
	fmt.Printf("%v", iplist)
}
