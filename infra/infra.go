package infra

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	Scheme              = "https"
	IPProtocol          = "tcp"
	ApiServerPortRange  = "6443/6443"
	SshPortRange        = "22/22"
	SourceCidrIP        = "0.0.0.0/0"
	CidrBlock           = "172.16.0.0/24"
	Policy              = "accept"
	DestinationResource = "InstanceType"
	InstanceChargeType  = "PostPaid"
	ImageID             = "centos_7_9_x64_20G_alibase_20210128.vhd"
	AccessKey           = "ACCESSKEYID"
	AccessSecret        = "ACCESSKEYSECRET"
	Product             = "product"
	Role                = "role"
	Master              = "master"
	Node                = "node"
	Stopped             = "Stopped"
	AvaibleTypeStatus   = "WithStock"
	Bandwidth           = "100"
	Digits              = "0123456789"
	Specials            = "~=+%^*/()[]{}/!@#$?|"
	Letter              = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	PasswordLength      = 16
	DataCategory        = "cloud_ssd"
	AliDomain           = "sea.aliyun.com/"
	AliPorvider         = "ALI_CLOUD"
	EipID               = AliDomain + "EipID"
	Master0ID           = AliDomain + "Master0ID"
	Master0InternalIP   = AliDomain + "Master0InternalIP"
	VpcID               = AliDomain + "VpcID"
	VSwitchID           = AliDomain + "VSwitchID"
	SecurityGroupID     = AliDomain + "SecurityGroupID"
	Eip                 = AliDomain + "ClusterEIP"
	RegionID            = "RegionID"
	AliRegionID         = AliDomain + RegionID
	DefaultReigonID     = "cn-chengdu"
	AliCloudEssd        = "cloud_essd"
	TryTimes            = 10
	TrySleepTime        = time.Second
	JustGetInstanceInfo = ""
)

type Interface interface {
	// apply iaas resources and save metadata info like vpc instance id to cluster status
	// https://github.com/fanux/sealgate/tree/master/cloud
	Apply(cluster *v1.Cluster) error
}

type Instance struct {
	CPU              int
	Memory           int
	InstanceID       string
	PrimaryIPAddress string
}

type Config struct {
	AccessKey    string
	AccessSecret string
	RegionID     string
}

type DefaultProvider struct {
	Scheme              string
	Config              Config
	EcsClient           *ecs.Client
	VpcClient           *vpc.Client
	Cluster             v1.Cluster
	Masters             []Instance
	Nodes               []Instance
	IPProtocol          string // "tcp"
	CidrBlock           string // "172.16.0.0/24"
	SshPortRange        string // "22/22"
	ApiServerPortRange  string // "6443/6443"
	SourceCidrIP        string // "0.0.0.0/0"
	Policy              string // "accept"
	DestinationResource string // "InstanceType"
	InstanceChargeType  string // "PostPaid"
	ImageID             string // "centos_7_9_x64_20G_alibase_20210128.vhd"
	MasterInstanceCount int
	NodeInstanceCount   int
	MasterInstanceType  string
	NodeInstanceType    string
	ProductTag          string
	ZoneID              string
	VpcID               string
	SecurityGroupID     string
	VSwitchID           string
	SpotStrategy        string
}

func GetAKSKFromEnv(config *Config) error {
	config.AccessKey = os.Getenv(AccessKey)
	config.AccessSecret = os.Getenv(AccessSecret)
	config.RegionID = os.Getenv(RegionID)
	if config.RegionID == "" {
		config.RegionID = DefaultReigonID
	}
	if config.AccessKey == "" || config.AccessSecret == "" || config.RegionID == "" {
		return fmt.Errorf("please set accessKey and accessKeySecret ENV, example: export ACCESSKEYID=xxx export ACCESSKEYSECRET=xxx , how to get AK SK: https://ram.console.aliyun.com/manage/ak")
	}
	return nil
}

func NewDefaultProvider(cluster *v1.Cluster) Interface {
	switch cluster.Spec.Provider {
	case AliPorvider:
		config := new(Config)
		err := GetAKSKFromEnv(config)
		if err != nil {
			logger.Error(err)
			return nil
		}
		defaultProvider := new(DefaultProvider)
		defaultProvider.Config = *config
		defaultProvider.NewClient()
		defaultProvider.InstanceChargeType = InstanceChargeType
		defaultProvider.DestinationResource = DestinationResource
		defaultProvider.Scheme = Scheme
		defaultProvider.IPProtocol = IPProtocol
		defaultProvider.SourceCidrIP = SourceCidrIP
		defaultProvider.CidrBlock = CidrBlock
		defaultProvider.ImageID = ImageID
		defaultProvider.Policy = Policy
		defaultProvider.VpcID = cluster.Annotations[VpcID]
		defaultProvider.VSwitchID = cluster.Annotations[VSwitchID]
		defaultProvider.SecurityGroupID = cluster.Annotations[SecurityGroupID]
		return defaultProvider
	default:
		return nil
	}

}

func (d *DefaultProvider) NewClient() error {
	ecsClient, err := ecs.NewClientWithAccessKey(d.Config.RegionID, d.Config.AccessKey, d.Config.AccessSecret)
	vpcClient, err := vpc.NewClientWithAccessKey(d.Config.RegionID, d.Config.AccessKey, d.Config.AccessSecret)

	if err != nil {
		return fmt.Errorf("create ali client failed")

	}
	d.EcsClient = ecsClient
	d.VpcClient = vpcClient
	return nil
}

func (d *DefaultProvider) GetZoneID() error {
	request := ecs.CreateDescribeZonesRequest()
	request.Scheme = d.Scheme
	request.InstanceChargeType = d.InstanceChargeType
	request.SpotStrategy = d.SpotStrategy
	response := ecs.CreateDescribeZonesResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	if len(response.Zones.Zone) == 0 {
		return fmt.Errorf("Not available ZoneID ")
	}
	d.ZoneID = response.Zones.Zone[0].ZoneId
	return nil
}

func (d *DefaultProvider) CreateVPC(cluster *v1.Cluster) error {
	request := ecs.CreateCreateVpcRequest()
	request.Scheme = d.Scheme
	request.RegionId = d.Config.RegionID
	//response, err := d.Client.CreateVpc(request)
	response := ecs.CreateCreateVpcResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	d.VpcID = response.VpcId
	cluster.Annotations[VpcID] = d.VpcID
	cluster.Annotations[AliRegionID] = d.Config.RegionID
	return nil
}

func (d DefaultProvider) AuthorizeSecurityGroup(securityGroupId, portRange string) bool {
	request := ecs.CreateAuthorizeSecurityGroupRequest()
	request.Scheme = d.Scheme
	request.SecurityGroupId = securityGroupId
	request.IpProtocol = d.IPProtocol
	request.PortRange = portRange
	request.SourceCidrIp = d.SourceCidrIP
	request.Policy = d.Policy

	//response, err := d.Client.AuthorizeSecurityGroup(request)
	response := ecs.CreateAuthorizeSecurityGroupResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		logger.Error("%v", err)
		return false
	}
	return response.BaseResponse.IsSuccess()
}

func (d *DefaultProvider) CreateSecurityGroup(cluster *v1.Cluster) error {
	request := ecs.CreateCreateSecurityGroupRequest()
	request.Scheme = d.Scheme
	request.RegionId = d.Config.RegionID
	request.VpcId = d.VpcID
	//response, err := d.Client.CreateSecurityGroup(request)
	response := ecs.CreateCreateSecurityGroupResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}

	if !d.AuthorizeSecurityGroup(response.SecurityGroupId, SshPortRange) {
		return fmt.Errorf("authorize securitygroup ssh port failed")
	}
	if !d.AuthorizeSecurityGroup(response.SecurityGroupId, ApiServerPortRange) {
		return fmt.Errorf("authorize securitygroup apiserver port failed")
	}
	d.SecurityGroupID = response.SecurityGroupId
	cluster.Annotations[SecurityGroupID] = d.SecurityGroupID
	return nil
}

func (d *DefaultProvider) CreateVSwitch(cluster *v1.Cluster) error {
	err := d.GetZoneID()
	if err != nil {
		return err
	}
	request := ecs.CreateCreateVSwitchRequest()
	request.Scheme = d.Scheme
	request.ZoneId = d.ZoneID
	request.CidrBlock = d.CidrBlock
	request.VpcId = d.VpcID
	request.RegionId = d.Config.RegionID
	//response, err := d.Client.CreateVSwitch(request)
	response := ecs.CreateCreateVSwitchResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	d.VSwitchID = response.VSwitchId
	cluster.Annotations[VSwitchID] = d.VSwitchID
	return nil
}

func (d *DefaultProvider) GetAvailableResource(cores int, memory float64) (instanceType string, err error) {
	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = d.Scheme
	request.RegionId = d.Config.RegionID
	request.DestinationResource = d.DestinationResource
	request.InstanceChargeType = d.InstanceChargeType
	request.Cores = requests.NewInteger(cores)
	request.Memory = requests.NewFloat(memory)

	//response, err := d.Client.DescribeAvailableResource(request)
	response := ecs.CreateDescribeAvailableResourceResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return "", err
	}

	if len(response.AvailableZones.AvailableZone) < 1 {
		return "", fmt.Errorf("resources not find")
	}
	for _, f := range response.AvailableZones.AvailableZone[0].AvailableResources.AvailableResource {
		for _, r := range f.SupportedResources.SupportedResource {
			if r.StatusCategory == AvaibleTypeStatus {
				return r.Value, nil
			}
		}
	}
	return "", nil
}

func (d *DefaultProvider) RunInstances(clusterHosts v1.Hosts, count int, instanceRole, clusterName, password string) error {
	instances := clusterHosts
	instancesCPU, _ := strconv.Atoi(clusterHosts.CPU)
	instancesMemory, _ := strconv.ParseFloat(clusterHosts.Memory, 64)
	systemDiskSize := clusterHosts.SystemDisk
	instanceType, err := d.GetAvailableResource(instancesCPU, instancesMemory)
	amount := count

	tag := make(map[string]string)
	tag[Product] = clusterName
	tag[Role] = instanceRole
	instancesTag := CreateInstanceTag(tag)

	dataDisks := instances.DataDisks
	datadisk := CreateInstanceDataDisk(dataDisks)

	request := ecs.CreateRunInstancesRequest()
	request.Scheme = d.Scheme
	request.ImageId = d.ImageID
	request.Password = password
	request.InstanceType = instanceType
	request.SecurityGroupId = d.SecurityGroupID
	request.VSwitchId = d.VSwitchID
	request.SystemDiskSize = systemDiskSize
	request.SystemDiskCategory = DataCategory
	request.DataDisk = &datadisk
	request.Amount = requests.NewInteger(amount)
	request.Tag = &instancesTag

	//response, err := d.Client.RunInstances(request)
	response := ecs.CreateRunInstancesResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	if !response.IsSuccess() {
		return fmt.Errorf("creat instance failed")
	}
	return nil
}

func CreateInstanceTag(tags map[string]string) (instanceTags []ecs.RunInstancesTag) {
	for k, v := range tags {
		instanceTags = append(instanceTags, ecs.RunInstancesTag{Key: k, Value: v})
	}
	return
}

func (d *DefaultProvider) Retry(tryTimes int, trySleepTime time.Duration, action func() error) error {
	var err error
	for i := 0; i < tryTimes; i++ {
		err = action()
		if err == nil {
			return nil
		}
		time.Sleep(trySleepTime * time.Duration(2*i+1))
	}
	return fmt.Errorf("retry action timeout: %v", err)
}

func (d *DefaultProvider) TryGetInstance(request *ecs.DescribeInstancesRequest, response *ecs.DescribeInstancesResponse, expectCount int) error {
	return d.Retry(TryTimes, TrySleepTime, func() error {
		err := d.EcsClient.DoAction(request, response)
		if err != nil {
			return err
		}
		instances := response.Instances.Instance
		if expectCount != -1 {
			if len(instances) != expectCount {
				return errors.New("The number of instances is not as expected")
			}
			for _, instance := range instances {
				if instance.NetworkInterfaces.NetworkInterface[0].PrimaryIpAddress == "" {
					return errors.New("PrimaryIpAddress cannt nob be nil")
				}
			}
		}

		return nil
	})
}

func (d *DefaultProvider) RetryEcsRequest(request requests.AcsRequest, response responses.AcsResponse) error {
	return d.Retry(TryTimes, TrySleepTime, func() error {
		err := d.EcsClient.DoAction(request, response)
		if err != nil {
			return err
		}
		return nil
	})
}

func (d *DefaultProvider) RetryVpcRequest(request requests.AcsRequest, response responses.AcsResponse) error {
	return d.Retry(TryTimes, TrySleepTime, func() error {
		err := d.VpcClient.DoAction(request, response)
		if err != nil {
			return err
		}
		return nil
	})
}

func CreateDescribeInstancesTag(tags map[string]string) (instanceTags []ecs.DescribeInstancesTag) {
	for k, v := range tags {
		instanceTags = append(instanceTags, ecs.DescribeInstancesTag{Key: k, Value: v})
	}
	return
}

func CreateInstanceDataDisk(dataDisks []string) (instanceDisks []ecs.RunInstancesDataDisk) {
	for _, v := range dataDisks {
		instanceDisks = append(instanceDisks,
			ecs.RunInstancesDataDisk{Size: v, Category: AliCloudEssd})
	}
	return
}

func (d *DefaultProvider) DeleteInstances(instanceIDs []string) error {
	request := ecs.CreateDeleteInstancesRequest()
	request.Scheme = d.Scheme
	request.InstanceId = &instanceIDs
	request.Force = requests.NewBoolean(true)
	//_, err := d.Client.DeleteInstances(request)
	response := ecs.CreateDeleteInstancesResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		return fmt.Errorf("delete instance failed %v , error :%v", instanceIDs, err)
	}
	return nil
}

func (d *DefaultProvider) DeleteVSwitch(vSwitchID string) error {
	request := ecs.CreateDeleteVSwitchRequest()
	request.Scheme = d.Scheme
	request.VSwitchId = vSwitchID

	//response, err := d.Client.DeleteVSwitch(request)
	response := ecs.CreateDeleteVSwitchResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) DeleteSecurityGroup(securityGroupID string) error {
	request := ecs.CreateDeleteSecurityGroupRequest()
	request.Scheme = d.Scheme
	request.SecurityGroupId = securityGroupID

	//response, err := d.Client.DeleteSecurityGroup(request)
	response := ecs.CreateDeleteSecurityGroupResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) DeleteVPC(vpcID string) error {
	request := ecs.CreateDeleteVpcRequest()
	request.Scheme = d.Scheme
	request.VpcId = vpcID

	//response, err := d.Client.DeleteVpc(request)
	response := ecs.CreateDeleteVpcResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) BindEipForMaster0(cluster *v1.Cluster, master0 Instance) error {
	eIP, eIPID, err := d.AllocateEipAddress()
	if err != nil {
		return err
	}
	err = d.AssociateEipAddress(master0.InstanceID, eIPID)
	if err != nil {
		return err
	}
	cluster.Annotations[Eip] = eIP
	cluster.Annotations[EipID] = eIPID
	cluster.Annotations[Master0ID] = master0.InstanceID
	cluster.Annotations[Master0InternalIP] = master0.PrimaryIPAddress
	return nil
}

func (d *DefaultProvider) AllocateEipAddress() (eIP, eIPID string, err error) {
	request := ecs.CreateAllocateEipAddressRequest()
	request.Scheme = d.Scheme
	request.Bandwidth = Bandwidth
	//response, err := d.Client.AllocateEipAddress(request)
	response := ecs.CreateAllocateEipAddressResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return "", "", err
	}
	return response.EipAddress, response.AllocationId, nil
}

func (d *DefaultProvider) AssociateEipAddress(instanceID, eipID string) error {
	request := ecs.CreateAssociateEipAddressRequest()
	request.Scheme = d.Scheme
	request.InstanceId = instanceID
	request.AllocationId = eipID

	//response, err := d.Client.AssociateEipAddress(request)
	response := ecs.CreateAssociateEipAddressResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) UnassociateEipAddress(eipID string) error {
	request := vpc.CreateUnassociateEipAddressRequest()
	request.Scheme = d.Scheme
	request.AllocationId = eipID
	request.Force = requests.NewBoolean(true)
	//response, err := d.Client.UnassociateEipAddress(request)
	response := vpc.CreateUnassociateEipAddressResponse()
	return d.RetryVpcRequest(request, response)
}

func (d *DefaultProvider) ReleaseEipAddress(eipID string) error {
	err := d.UnassociateEipAddress(eipID)
	if err != nil {
		return err
	}
	request := ecs.CreateReleaseEipAddressRequest()
	request.Scheme = d.Scheme
	request.AllocationId = eipID
	//response, err := d.Client.ReleaseEipAddress(request)
	response := ecs.CreateReleaseEipAddressResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) GetInstanceStatus(instanceID string) (instanceStatus string, err error) {
	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = d.Scheme
	request.InstanceId = &[]string{instanceID}

	//response, err := d.Client.DescribeInstanceStatus(request)
	response := ecs.CreateDescribeInstanceStatusResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return "", fmt.Errorf("get instance status failed %v , error :%v", instanceID, err)
	}
	if len(response.InstanceStatuses.InstanceStatus) == 0 {
		return "", fmt.Errorf("instance list is empty")
	}
	return response.InstanceStatuses.InstanceStatus[0].Status, nil
}

func (d *DefaultProvider) PoweroffInstance(instanceID string) error {
	request := ecs.CreateStopInstancesRequest()
	request.Scheme = d.Scheme
	request.InstanceId = &[]string{instanceID}

	//_, err := d.Client.StopInstances(request)
	response := ecs.CreateStopInstancesResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) StartInstance(instanceID string) error {
	request := ecs.CreateStartInstanceRequest()
	request.Scheme = d.Scheme
	request.InstanceId = instanceID

	//_, err := d.Client.StartInstance(request)
	response := ecs.CreateStartInstanceResponse()
	return d.RetryEcsRequest(request, response)
}

func (d *DefaultProvider) ChangeInstanceType(instanceID, cpu, memory string) error {
	cpu_int, err := strconv.Atoi(cpu)
	if err != nil {
		return err
	}
	memory_float, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return err
	}
	instanceStatus, err := d.GetInstanceStatus(instanceID)
	if err != nil {
		return err
	}
	if instanceStatus != Stopped {
		err = d.PoweroffInstance(instanceID)
		if err != nil {
			return err
		}
	}
	expectInstanceType, err := d.GetAvailableResource(cpu_int, memory_float)
	if err != nil {
		return err
	}

	request := ecs.CreateModifyInstanceSpecRequest()
	request.Scheme = d.Scheme
	request.InstanceId = instanceID
	request.InstanceType = expectInstanceType
	//_, err = d.Client.ModifyInstanceSpec(request)
	response := ecs.CreateModifyInstanceSpecResponse()
	err = d.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	return d.StartInstance(instanceID)
}

func (d *DefaultProvider) GetInstancesInfo(productName, instancesRole, vSwitchID, securityGroupID, expectCount string) (instances []Instance, err error) {
	tag := make(map[string]string)
	var count int
	if expectCount == "" {
		count = -1
	} else {
		count, err = strconv.Atoi(expectCount)
	}

	if err != nil {
		return nil, err
	}
	tag[Product] = productName
	tag[Role] = instancesRole
	instancesTags := CreateDescribeInstancesTag(tag)
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = d.Scheme
	request.RegionId = d.Config.RegionID
	request.VSwitchId = vSwitchID
	request.SecurityGroupId = securityGroupID
	request.Tag = &instancesTags
	//response, err := d.Client.DescribeInstances(request)
	response := ecs.CreateDescribeInstancesResponse()

	err = d.TryGetInstance(request, response, count)
	if err != nil {
		return nil, err
	}

	for _, instance := range response.Instances.Instance {
		instances = append(instances,
			Instance{
				CPU:              instance.Cpu,
				Memory:           instance.Memory / 1024,
				InstanceID:       instance.InstanceId,
				PrimaryIPAddress: instance.NetworkInterfaces.NetworkInterface[0].PrimaryIpAddress})
	}
	return
}

func (d *DefaultProvider) InputIPlist(hosts *v1.Hosts, clusterName, instanceRole string) (iplist []string, err error) {
	var ipList []string

	instances, err := d.GetInstancesInfo(clusterName, instanceRole, d.VSwitchID, d.SecurityGroupID, hosts.Count)
	if err != nil {
		return nil, err
	}
	for _, instance := range instances {
		ipList = append(ipList, instance.PrimaryIPAddress)
	}
	return ipList, nil
}

func (d *DefaultProvider) CreatePassword(cluster *v1.Cluster) {
	rand.Seed(time.Now().UnixNano())
	digits := Digits
	specials := Specials
	letter := Letter
	all := digits + specials + letter
	length := PasswordLength
	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	buf[1] = specials[rand.Intn(len(specials))]
	for i := 2; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})
	cluster.Spec.SSH.Passwd = string(buf)
}

func (d *DefaultProvider) ReconcileIntances(cluster *v1.Cluster, instanceRole string, isInit bool) error {
	var hosts *v1.Hosts
	var instances []Instance
	switch instanceRole {
	case Master:
		hosts = &cluster.Spec.Masters
		if hosts.Count == "" {
			return errors.New("master count not set")
		}
	case Node:
		hosts = &cluster.Spec.Nodes
		if hosts.Count == "" {
			return nil
		}
	}

	i, err := strconv.Atoi(hosts.Count)
	if err != nil {
		return err
	}
	if !isInit {
		instances, err = d.GetInstancesInfo(cluster.Name, instanceRole, cluster.Annotations[VSwitchID], cluster.Annotations[SecurityGroupID], JustGetInstanceInfo)
	}
	if err != nil {
		return err
	}
	if len(instances) < i {
		err = d.RunInstances(*hosts, i-len(instances), instanceRole, cluster.Name, cluster.Spec.SSH.Passwd)
		if err != nil {
			return err
		}
		ipList, err := d.InputIPlist(hosts, cluster.Name, instanceRole)
		if err != nil {
			return err
		}
		hosts.IPList = utils.AppendIPList(hosts.IPList, ipList)

	} else if len(instances) > i {
		var deleteInstancesIDs []string
		var count int
		for _, instance := range instances {
			if instance.InstanceID != cluster.Annotations[Master0ID] {
				deleteInstancesIDs = append(deleteInstancesIDs, instance.InstanceID)
				count += 1
			}
			if count == (len(instances) - i) {
				break
			}
		}

		err = d.DeleteInstances(deleteInstancesIDs)
		if err != nil {
			return err
		}
		ipList, err := d.InputIPlist(hosts, cluster.Name, instanceRole)
		if err != nil {
			return err
		}
		hosts.IPList = utils.ReduceIPList(hosts.IPList, ipList)
	}
	cpu, err := strconv.Atoi(hosts.CPU)
	memory, _ := strconv.Atoi(hosts.Memory)
	for _, instance := range instances {
		if instance.CPU != cpu || instance.Memory != memory {
			err = d.ChangeInstanceType(instance.InstanceID, hosts.CPU, hosts.Memory)
			if err != nil {
				return err
			}
		}
	}

	logger.Info("reconcile %s instances success %v ", instanceRole, hosts.IPList)
	return nil
}

func (d *DefaultProvider) ClearCluster(cluster *v1.Cluster) error {
	var instanceIDs = []string{}

	//Release Eip
	if eipID, ok := cluster.Annotations[EipID]; ok && eipID != "" {
		err := d.ReleaseEipAddress(eipID)
		if err != nil {
			logger.Error("Release EIP failed err: %v", err)
		}
	}

	//Get instancesid for delete instances
	instanceMasters, err := d.GetInstancesInfo(cluster.Name, Master, cluster.Annotations[VSwitchID], cluster.Annotations[SecurityGroupID], JustGetInstanceInfo)
	if err != nil {
		return err
	}
	for _, instance := range instanceMasters {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	instanceNodes, err := d.GetInstancesInfo(cluster.Name, Node, cluster.Annotations[VSwitchID], cluster.Annotations[SecurityGroupID], JustGetInstanceInfo)
	if err != nil {
		return err
	}
	for _, instance := range instanceNodes {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}

	//Delete instance
	if len(instanceIDs) != 0 {
		err := d.DeleteInstances(instanceIDs)
		if err != nil {
			logger.Error("delete instances failed %v", err)
		} else {
			logger.Info("delete instances success")
		}
	}
	//Delete vSwitch
	if vSwitchID, ok := cluster.Annotations[VSwitchID]; ok && vSwitchID != "" {
		err := d.DeleteVSwitch(vSwitchID)
		if err != nil {
			logger.Error("delete VSwitch faile err: %s", err)
		} else {
			logger.Info("delete VSwitch: %s Success", vSwitchID)
		}
	}
	//Delete SecurityGroup
	if securityGroupID, ok := cluster.Annotations[SecurityGroupID]; ok && securityGroupID != "" {
		err := d.DeleteSecurityGroup(securityGroupID)
		if err != nil {
			logger.Error("delete SecurityGroup faile err: %s", err)
		} else {
			logger.Info("delete SecurityGroup: %s Success", securityGroupID)
		}

	}
	//Delete VPC
	if vpcID, ok := cluster.Annotations[VpcID]; ok && vpcID != "" {
		err := d.DeleteVPC(vpcID)
		if err != nil {
			logger.Error("delete VPC faile err: %s", err)
		} else {
			logger.Info("delete VPC: %s Success", vpcID)
		}

	}
	return nil
}

func (d *DefaultProvider) GetVPC() ([]string, error) {
	var vpcIDs = []string{}
	request := ecs.CreateDescribeVpcsRequest()
	request.Scheme = d.Scheme
	//response, err := d.Client.DescribeVpcs(request)
	response := ecs.CreateDescribeVpcsResponse()
	err := d.RetryEcsRequest(request, response)
	if err != nil {
		return nil, err
	}
	for _, vpc := range response.Vpcs.Vpc {
		vpcIDs = append(vpcIDs, vpc.VpcId)
	}

	return vpcIDs, nil
}

func (d *DefaultProvider) Reconcile(cluster *v1.Cluster) error {
	var isInit bool
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	if cluster.Annotations[VpcID] == "" {
		isInit = true
	}
	if cluster.DeletionTimestamp != nil {
		logger.Info("DeletionTimestamp not nil Clear Cluster")
		return d.ClearCluster(cluster)
	}

	if vpcID, ok := cluster.Annotations[VpcID]; !ok || vpcID == "" {
		err := d.CreateVPC(cluster)
		if err != nil {
			return err
		}
		logger.Info("create VPC success VPCID: %s", cluster.Annotations[VpcID])
	}

	if vswitchID, ok := cluster.Annotations[VSwitchID]; !ok || vswitchID == "" {
		err := d.CreateVSwitch(cluster)
		if err != nil {
			return err
		}
		logger.Info("create VSwitch success VSwitchID: %s", cluster.Annotations[VSwitchID])
	}

	if securityGroupID, ok := cluster.Annotations[SecurityGroupID]; !ok || securityGroupID == "" {
		err := d.CreateSecurityGroup(cluster)
		if err != nil {
			return err
		}
		logger.Info("create SecurityGroup success SecurityGroupID: %s", cluster.Annotations[SecurityGroupID])
	}

	if cluster.Spec.SSH.Passwd == "" {
		// Create ssh password
		d.CreatePassword(cluster)
	}

	err := d.ReconcileIntances(cluster, Master, isInit)
	if err != nil {
		return err
	}

	err = d.ReconcileIntances(cluster, Node, isInit)
	if err != nil {
		return err
	}

	if eIP, ok := cluster.Annotations[Eip]; !ok || eIP == "" {
		var master0 = Instance{}
		masters, err := d.GetInstancesInfo(cluster.Name,
			Master, cluster.Annotations[VSwitchID],
			cluster.Annotations[SecurityGroupID], cluster.Spec.Masters.Count)
		if err != nil {
			return err
		}
		if len(masters) != 0 {
			master0 = masters[0]
		}
		// bound EIP
		err = d.BindEipForMaster0(cluster, master0)
		if err != nil {
			return err
		}
		logger.Info("bind eip(%s) to master0(%s) success ", cluster.Annotations[Eip], cluster.Annotations[Master0InternalIP])
	}
	return nil
}

func (d *DefaultProvider) Apply(cluster *v1.Cluster) error {
	return d.Reconcile(cluster)
}
