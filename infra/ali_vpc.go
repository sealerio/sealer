package infra

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

type VpcManager struct {
	Config Config
	Client *vpc.Client
}

func (a *AliProvider) RetryVpcRequest(request requests.AcsRequest, response responses.AcsResponse) error {
	return Retry(TryTimes, TrySleepTime, func() error {
		err := a.VpcClient.DoAction(request, response)
		if err != nil {
			return err
		}
		return nil
	})
}

func (a *AliProvider) CreateVPC() error {
	request := vpc.CreateCreateVpcRequest()
	request.Scheme = Scheme
	request.RegionId = a.Config.RegionID
	//response, err := d.Client.CreateVpc(request)
	response := vpc.CreateCreateVpcResponse()
	err := a.RetryVpcRequest(request, response)
	if err != nil {
		return err
	}
	a.Cluster.Annotations[VpcID] = response.VpcId
	a.Cluster.Annotations[AliRegionID] = a.Config.RegionID
	return nil
}

func (a *AliProvider) DeleteVPC() error {
	request := vpc.CreateDeleteVpcRequest()
	request.Scheme = Scheme
	request.VpcId = a.Cluster.Annotations[VpcID]

	//response, err := d.Client.DeleteVpc(request)
	response := vpc.CreateDeleteVpcResponse()
	return a.RetryVpcRequest(request, response)
}

func (a *AliProvider) CreateVSwitch() error {
	request := vpc.CreateCreateVSwitchRequest()
	request.Scheme = Scheme
	request.ZoneId = a.Cluster.Annotations[ZoneID]
	request.CidrBlock = CidrBlock
	request.VpcId = a.Cluster.GetAnnotationsByKey(VpcID)
	request.RegionId = a.Config.RegionID
	//response, err := d.Client.CreateVSwitch(request)
	response := vpc.CreateCreateVSwitchResponse()
	err := a.RetryVpcRequest(request, response)
	if err != nil {
		return err
	}
	a.Cluster.Annotations[VSwitchID] = response.VSwitchId
	return nil
}

func (a *AliProvider) DeleteVSwitch() error {
	request := vpc.CreateDeleteVSwitchRequest()
	request.Scheme = Scheme
	request.VSwitchId = a.Cluster.Annotations[VSwitchID]

	//response, err := d.Client.DeleteVSwitch(request)
	response := vpc.CreateDeleteVSwitchResponse()
	return a.RetryVpcRequest(request, response)
}

func (a *AliProvider) GetZoneID() error {
	request := vpc.CreateDescribeZonesRequest()
	request.Scheme = Scheme
	response := vpc.CreateDescribeZonesResponse()
	err := a.RetryVpcRequest(request, response)
	if err != nil {
		return err
	}
	if len(response.Zones.Zone) == 0 {
		return fmt.Errorf("Not available ZoneID ")
	}
	a.Cluster.Annotations[ZoneID] = response.Zones.Zone[0].ZoneId

	return nil
}

func (a *AliProvider) BindEipForMaster0() error {
	instances, err := a.GetInstancesInfo(Master, JustGetInstanceInfo)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return errors.New("can not find master0 ")
	}
	master0 := instances[0]
	eIP, eIPID, err := a.AllocateEipAddress()
	if err != nil {
		return err
	}
	err = a.AssociateEipAddress(master0.InstanceID, eIPID)
	if err != nil {
		return err
	}
	a.Cluster.Annotations[Eip] = eIP
	a.Cluster.Annotations[EipID] = eIPID
	a.Cluster.Annotations[Master0ID] = master0.InstanceID
	a.Cluster.Annotations[Master0InternalIP] = master0.PrimaryIPAddress
	return nil
}

func (a *AliProvider) AllocateEipAddress() (eIP, eIPID string, err error) {
	request := vpc.CreateAllocateEipAddressRequest()
	request.Scheme = Scheme
	request.Bandwidth = Bandwidth
	//response, err := d.Client.AllocateEipAddress(request)
	response := vpc.CreateAllocateEipAddressResponse()
	err = a.RetryVpcRequest(request, response)
	if err != nil {
		return "", "", err
	}
	return response.EipAddress, response.AllocationId, nil
}

func (a *AliProvider) AssociateEipAddress(instanceID, eipID string) error {
	request := vpc.CreateAssociateEipAddressRequest()
	request.Scheme = Scheme
	request.InstanceId = instanceID
	request.AllocationId = eipID

	//response, err := d.Client.AssociateEipAddress(request)
	response := vpc.CreateAssociateEipAddressResponse()
	return a.RetryVpcRequest(request, response)
}

func (a *AliProvider) UnassociateEipAddress() error {
	request := vpc.CreateUnassociateEipAddressRequest()
	request.Scheme = Scheme
	request.AllocationId = a.Cluster.Annotations[EipID]
	request.Force = requests.NewBoolean(true)
	//response, err := d.Client.UnassociateEipAddress(request)
	response := vpc.CreateUnassociateEipAddressResponse()
	return a.RetryVpcRequest(request, response)
}

func (a *AliProvider) ReleaseEipAddress() error {
	err := a.UnassociateEipAddress()
	if err != nil {
		return err
	}
	request := vpc.CreateReleaseEipAddressRequest()
	request.Scheme = Scheme
	request.AllocationId = a.Cluster.Annotations[EipID]
	//response, err := d.Client.ReleaseEipAddress(request)
	response := vpc.CreateReleaseEipAddressResponse()
	return a.RetryVpcRequest(request, response)
}
