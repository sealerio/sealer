package infra

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Instance struct {
	CPU              int
	Memory           int
	InstanceID       string
	PrimaryIPAddress string
}

type EcsManager struct {
	config Config
	client *ecs.Client
}

func (a *AliProvider) RetryEcsRequest(request requests.AcsRequest, response responses.AcsResponse) error {
	return Retry(TryTimes, TrySleepTime, func() error {
		err := a.EcsClient.DoAction(request, response)
		if err != nil {
			return err
		}
		return nil
	})
}

func (a *AliProvider) TryGetInstance(request *ecs.DescribeInstancesRequest, response *ecs.DescribeInstancesResponse, expectCount int) error {
	return Retry(TryTimes, TrySleepTime, func() error {
		err := a.EcsClient.DoAction(request, response)
		if err != nil {
			return err
		}
		instances := response.Instances.Instance
		if expectCount != -1 {
			if len(instances) != expectCount {
				return errors.New("the number of instances is not as expected")
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

func (a *AliProvider) InputIPlist(instanceRole string) (iplist []string, err error) {
	var ipList []string
	var hosts *v1.Hosts
	switch instanceRole {
	case Master:
		hosts = &a.Cluster.Spec.Masters
	case Node:
		hosts = &a.Cluster.Spec.Nodes
	}
	if hosts == nil {
		return nil, err
	}
	instances, err := a.GetInstancesInfo(instanceRole, hosts.Count)
	if err != nil {
		return nil, err
	}
	for _, instance := range instances {
		ipList = append(ipList, instance.PrimaryIPAddress)
	}
	return ipList, nil
}

func (a *AliProvider) CreatePassword() {
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
	a.Cluster.Spec.SSH.Passwd = string(buf)
}

func (a *AliProvider) GetInstanceStatus(instanceID string) (instanceStatus string, err error) {
	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = Scheme
	request.InstanceId = &[]string{instanceID}

	//response, err := d.Client.DescribeInstanceStatus(request)
	response := ecs.CreateDescribeInstanceStatusResponse()
	err = a.RetryEcsRequest(request, response)
	if err != nil {
		return "", fmt.Errorf("get instance status failed %v , error :%v", instanceID, err)
	}
	if len(response.InstanceStatuses.InstanceStatus) == 0 {
		return "", fmt.Errorf("instance list is empty")
	}
	return response.InstanceStatuses.InstanceStatus[0].Status, nil
}

func (a *AliProvider) PoweroffInstance(instanceID string) error {
	request := ecs.CreateStopInstancesRequest()
	request.Scheme = Scheme
	request.InstanceId = &[]string{instanceID}

	//_, err := d.Client.StopInstances(request)
	response := ecs.CreateStopInstancesResponse()
	return a.RetryEcsRequest(request, response)
}

func (a *AliProvider) StartInstance(instanceID string) error {
	request := ecs.CreateStartInstanceRequest()
	request.Scheme = Scheme
	request.InstanceId = instanceID

	//_, err := d.Client.StartInstance(request)
	response := ecs.CreateStartInstanceResponse()
	return a.RetryEcsRequest(request, response)
}

func (a *AliProvider) ChangeInstanceType(instanceID, cpu, memory string) error {
	cpuInt, err := strconv.Atoi(cpu)
	if err != nil {
		return err
	}
	memoryFloat, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return err
	}
	instanceStatus, err := a.GetInstanceStatus(instanceID)
	if err != nil {
		return err
	}
	if instanceStatus != Stopped {
		err = a.PoweroffInstance(instanceID)
		if err != nil {
			return err
		}
	}
	expectInstanceType, err := a.GetAvailableResource(cpuInt, memoryFloat)
	if err != nil {
		return err
	}

	request := ecs.CreateModifyInstanceSpecRequest()
	request.Scheme = Scheme
	request.InstanceId = instanceID
	request.InstanceType = expectInstanceType
	//_, err = d.Client.ModifyInstanceSpec(request)
	response := ecs.CreateModifyInstanceSpecResponse()
	err = a.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	return a.StartInstance(instanceID)
}

func (a *AliProvider) GetInstancesInfo(instancesRole, expectCount string) (instances []Instance, err error) {
	var count int
	tag := make(map[string]string)
	tag[Product] = a.Cluster.Name
	tag[Role] = instancesRole
	if expectCount == "" {
		count = -1
	} else {
		count, _ = strconv.Atoi(expectCount)
	}
	instancesTags := CreateDescribeInstancesTag(tag)
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = Scheme
	request.RegionId = a.Config.RegionID
	request.VSwitchId = a.Cluster.Annotations[VSwitchID]
	request.SecurityGroupId = a.Cluster.Annotations[SecurityGroupID]
	request.Tag = &instancesTags
	//response, err := d.Client.DescribeInstances(request)
	response := ecs.CreateDescribeInstancesResponse()
	err = a.TryGetInstance(request, response, count)
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

func (a *AliProvider) ReconcileIntances(instanceRole string) error {
	var hosts *v1.Hosts
	var instances []Instance
	var instancesIDs string
	switch instanceRole {
	case Master:
		hosts = &a.Cluster.Spec.Masters
		instancesIDs = a.Cluster.Annotations[AliMasterIDs]
		if hosts.Count == "" {
			return errors.New("master count not set")
		}
	case Node:
		hosts = &a.Cluster.Spec.Nodes
		instancesIDs = a.Cluster.Annotations[AliNodeIDs]
		if hosts.Count == "" {
			return nil
		}
	}
	if hosts == nil {
		return errors.New("hosts not set")
	}
	i, err := strconv.Atoi(hosts.Count)
	if err != nil {
		return err
	}
	if instancesIDs != "" {
		instances, err = a.GetInstancesInfo(instanceRole, JustGetInstanceInfo)
	}

	if err != nil {
		return err
	}
	if len(instances) < i {
		err = a.RunInstances(instanceRole, i-len(instances))
		if err != nil {
			return err
		}
		ipList, err := a.InputIPlist(instanceRole)
		if err != nil {
			return err
		}
		hosts.IPList = utils.AppendIPList(hosts.IPList, ipList)

	} else if len(instances) > i {
		var deleteInstancesIDs []string
		var count int
		for _, instance := range instances {
			if instance.InstanceID != a.Cluster.Annotations[Master0ID] {
				deleteInstancesIDs = append(deleteInstancesIDs, instance.InstanceID)
				count += 1
			}
			if count == (len(instances) - i) {
				break
			}
		}
		if len(deleteInstancesIDs) != 0 {
			a.Cluster.Annotations[ShouldBeDeleteInstancesIDs] = strings.Join(deleteInstancesIDs, ",")
			err = a.DeleteInstances()
			if err != nil {
				return err
			}
			a.Cluster.Annotations[ShouldBeDeleteInstancesIDs] = ""
		}

		ipList, err := a.InputIPlist(instanceRole)
		if err != nil {
			return err
		}
		hosts.IPList = utils.ReduceIPList(hosts.IPList, ipList)
	}
	cpu, err := strconv.Atoi(hosts.CPU)
	if err != nil {
		return err
	}
	memory, err := strconv.Atoi(hosts.Memory)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		if instance.CPU != cpu || instance.Memory != memory {
			err = a.ChangeInstanceType(instance.InstanceID, hosts.CPU, hosts.Memory)
			if err != nil {
				return err
			}
		}
	}

	logger.Info("reconcile %s instances success %v ", instanceRole, hosts.IPList)
	return nil
}

func (a *AliProvider) DeleteInstances() error {
	instanceIDs := strings.Split(a.Cluster.Annotations[ShouldBeDeleteInstancesIDs], ",")
	if len(instanceIDs) == 0 {
		return nil
	}
	request := ecs.CreateDeleteInstancesRequest()
	request.Scheme = Scheme
	request.InstanceId = &instanceIDs
	request.Force = requests.NewBoolean(true)
	//_, err := d.Client.DeleteInstances(request)
	response := ecs.CreateDeleteInstancesResponse()
	err := a.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}
	a.Cluster.Annotations[ShouldBeDeleteInstancesIDs] = ""
	return nil
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

func (a *AliProvider) GetAvailableResource(cores int, memory float64) (instanceType string, err error) {
	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = Scheme
	request.RegionId = a.Config.RegionID
	request.DestinationResource = DestinationResource
	request.InstanceChargeType = InstanceChargeType
	request.Cores = requests.NewInteger(cores)
	request.Memory = requests.NewFloat(memory)

	//response, err := d.Client.DescribeAvailableResource(request)
	response := ecs.CreateDescribeAvailableResourceResponse()
	err = a.RetryEcsRequest(request, response)
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

func (a *AliProvider) RunInstances(instanceRole string, count int) error {
	var hosts *v1.Hosts
	switch instanceRole {
	case Master:
		hosts = &a.Cluster.Spec.Masters
	case Node:
		hosts = &a.Cluster.Spec.Nodes
	}
	instances := hosts
	if instances == nil {
		return errors.New("host not set")
	}
	instancesCPU, _ := strconv.Atoi(instances.CPU)
	instancesMemory, _ := strconv.ParseFloat(instances.Memory, 64)
	systemDiskSize := instances.SystemDisk
	instanceType, err := a.GetAvailableResource(instancesCPU, instancesMemory)

	tag := make(map[string]string)
	tag[Product] = a.Cluster.Name
	tag[Role] = instanceRole
	instancesTag := CreateInstanceTag(tag)

	dataDisks := instances.DataDisks
	datadisk := CreateInstanceDataDisk(dataDisks)

	request := ecs.CreateRunInstancesRequest()
	request.Scheme = Scheme
	request.ImageId = ImageID
	request.Password = a.Cluster.Spec.SSH.Passwd
	request.InstanceType = instanceType
	request.SecurityGroupId = a.Cluster.GetAnnotationsByKey(SecurityGroupID)
	request.VSwitchId = a.Cluster.GetAnnotationsByKey(VSwitchID)
	request.SystemDiskSize = systemDiskSize
	request.SystemDiskCategory = DataCategory
	request.DataDisk = &datadisk
	request.Amount = requests.NewInteger(count)
	request.Tag = &instancesTag

	//response, err := d.Client.RunInstances(request)
	response := ecs.CreateRunInstancesResponse()
	err = a.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}

	instancesIDs := strings.Join(response.InstanceIdSets.InstanceIdSet, ",")
	switch instanceRole {
	case Master:
		a.Cluster.Annotations[AliMasterIDs] += instancesIDs
	case Node:
		a.Cluster.Annotations[AliNodeIDs] += instancesIDs
	}

	return nil
}

func (a *AliProvider) AuthorizeSecurityGroup(securityGroupId, portRange string) bool {
	request := ecs.CreateAuthorizeSecurityGroupRequest()
	request.Scheme = Scheme
	request.SecurityGroupId = securityGroupId
	request.IpProtocol = IPProtocol
	request.PortRange = portRange
	request.SourceCidrIp = SourceCidrIP
	request.Policy = Policy

	//response, err := d.Client.AuthorizeSecurityGroup(request)
	response := ecs.CreateAuthorizeSecurityGroupResponse()
	err := a.RetryEcsRequest(request, response)
	if err != nil {
		logger.Error("%v", err)
		return false
	}
	return response.BaseResponse.IsSuccess()
}

func (a *AliProvider) CreateSecurityGroup() error {
	request := ecs.CreateCreateSecurityGroupRequest()
	request.Scheme = Scheme
	request.RegionId = a.Config.RegionID
	request.VpcId = a.Cluster.GetAnnotationsByKey(VpcID)
	//response, err := d.Client.CreateSecurityGroup(request)
	response := ecs.CreateCreateSecurityGroupResponse()
	err := a.RetryEcsRequest(request, response)
	if err != nil {
		return err
	}

	if !a.AuthorizeSecurityGroup(response.SecurityGroupId, SshPortRange) {
		return fmt.Errorf("authorize securitygroup ssh port failed")
	}
	if !a.AuthorizeSecurityGroup(response.SecurityGroupId, ApiServerPortRange) {
		return fmt.Errorf("authorize securitygroup apiserver port failed")
	}
	a.Cluster.Annotations[SecurityGroupID] = response.SecurityGroupId
	return nil
}

func (a *AliProvider) DeleteSecurityGroup() error {

	request := ecs.CreateDeleteSecurityGroupRequest()
	request.Scheme = Scheme
	request.SecurityGroupId = a.Cluster.Annotations[SecurityGroupID]

	//response, err := d.Client.DeleteSecurityGroup(request)
	response := ecs.CreateDeleteSecurityGroupResponse()
	return a.RetryEcsRequest(request, response)
}
