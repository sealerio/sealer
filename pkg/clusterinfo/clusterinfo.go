// Copyright © 2022 Alibaba Group Holding Ltd.
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

package clusterinfo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/clusterinfo/api/types"
	"github.com/sealerio/sealer/pkg/clusterinfo/parse"
	v2 "github.com/sealerio/sealer/types/api/v2"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/ssh"
	strUtil "github.com/sealerio/sealer/utils/strings"
	"gopkg.in/yaml.v2"
)

func GetClusterInfo(cluster *v2.Cluster) (*types.ClusterInfoDetailed, error) {
	var ipList, ipNeedAddGetInfo []string
	for _, hosts := range cluster.Spec.Hosts {
		ipList = append(ipList, hosts.IPS...)
	}

	detailed := &types.ClusterInfoDetailed{}
	infoPath := common.GetClusterWorkClusterInfo(cluster.Name)
	if !osi.IsFileExist(infoPath) {
		data, err := ParseClusterInfo(ipList, cluster)
		if err != nil {
			return nil, err
		}

		err = writeToDisk(data, cluster)
		if err != nil {
			return nil, err
		}
		return detailed, nil
	}

	// load info disk cluster info, and compare its host ip,make sure is it contains all node info.
	b, err := ioutil.ReadFile(filepath.Clean(infoPath))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, detailed)
	if err != nil {
		return nil, err
	}

	for _, instanceInfo := range detailed.InstanceInfos {
		if !strUtil.NotIn(instanceInfo.PrivateIP, ipList) {
			continue
		}
		ipNeedAddGetInfo = append(ipNeedAddGetInfo, instanceInfo.PrivateIP)
	}

	if len(ipNeedAddGetInfo) == 0 {
		return detailed, nil
	}

	// if scale up node,get the node info.
	data, err := ParseClusterInfo(ipNeedAddGetInfo, cluster)
	if err != nil {
		return nil, err
	}

	detailed.InstanceInfos = append(detailed.InstanceInfos, data.InstanceInfos...)

	err = writeToDisk(detailed, cluster)
	if err != nil {
		return nil, err
	}

	return detailed, nil
}

func writeToDisk(detailed *types.ClusterInfoDetailed, cluster *v2.Cluster) error {
	clusterInfoStream, err := yaml.Marshal(detailed)
	if err != nil {
		return err
	}
	logger.Debug("get cluster info success %s", string(clusterInfoStream))
	err = ioutil.WriteFile(common.GetClusterWorkClusterInfo(cluster.Name), clusterInfoStream, common.FileMode0644)
	if err != nil {
		return err
	}
	return nil
}

func ParseClusterInfo(iplist []string, cluster *v2.Cluster) (*types.ClusterInfoDetailed, error) {
	detailed := &types.ClusterInfoDetailed{}
	scriptDir := common.GetClusterCheckScriptDir(cluster.Name)

	if err := parse.DumpScripts(scriptDir); err != nil {
		return nil, errors.Errorf("save scripts failed: %v", err)
	}

	checkerPath := parse.GetScriptPath(scriptDir, parse.ParseInstance.Name())
	defer func() {
		if err := os.RemoveAll(scriptDir); err != nil {
			logger.Error(err.Error())
		}
	}()

	for _, ipAddr := range iplist {
		ip := ipAddr
		sshClient, sshErr := ssh.NewStdoutSSHClient(ip, cluster)
		if sshErr != nil {
			return nil, sshErr
		}

		remoteScriptPath := parse.GetTmpScriptPath(parse.ParseInstance.Name())
		if err := sshClient.Copy(ip, checkerPath, remoteScriptPath); err != nil {
			return nil, errors.Errorf("copy file %s to %s:%s failed", checkerPath, ip, remoteScriptPath)
		}

		output, err := sshClient.Cmd(ip, fmt.Sprintf("chmod +x %s && bash %s %s", remoteScriptPath, remoteScriptPath, parse.ParseInstance.Params()))
		if err != nil {
			return nil, err
		}

		err = parseInstances(ip, output, detailed)
		if err != nil {
			return nil, err
		}
	}

	return detailed, nil
}

func parseInstances(host string, output []byte, info *types.ClusterInfoDetailed) (err error) {
	str := string(output)
	instanceInfo, err := parseInstanceInfo(str)
	if err != nil {
		return errors.Errorf("failed to parse command result %v failed", err)
	}

	instanceInfo.PrivateIP = IPFormat(host)
	info.InstanceInfos = append(info.InstanceInfos, *instanceInfo)

	return nil
}

func parseInstanceInfo(str string) (*types.InstanceInfo, error) {
	var instanceInfoExtended types.InstanceInfoExtended

	str, err := getBetweenStr(str, "##INSTANCE_INFO_BEGIN##", "##INSTANCE_INFO_END##")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(str), &instanceInfoExtended); err != nil {
		return nil, err
	}

	instanceInfo := instanceInfoExtended.InstanceInfo
	instanceInfo.Memory = (instanceInfo.Memory + 1024*1024 - 1) / (1024 * 1024)
	instanceInfo.TimeSyncStatus = instanceInfoExtended.TimeSyncStatus

	networkDevicesStr, err := base64.StdEncoding.DecodeString(instanceInfoExtended.NetworkDevicesStr)
	if err != nil {
		return nil, err
	}
	for _, netDevStr := range strings.Split(string(networkDevicesStr), "##SPLITER##") {
		netCard := parseNetCard(netDevStr)
		//skip lo,gw
		if netCard.Name == "lo" || netCard.Name == "gw" {
			continue
		}
		instanceInfo.NetworkCards = append(instanceInfo.NetworkCards, &netCard)
	}

	blockDevicesStr, err := base64.StdEncoding.DecodeString(instanceInfoExtended.BlockDevicesStr)
	if err != nil {
		return nil, err
	}
	systemDiskName := ""
	disks := make(map[string]*types.Disk)
	for _, blkDevStr := range strings.Split(string(blockDevicesStr), "##SPLITER##") {
		blkDev, parentName, err := parseBlkDev(blkDevStr)
		if err != nil {
			return nil, err
		}

		if isSystemDisk(blkDev) {
			if parentName != "" {
				systemDiskName = parentName
			} else {
				systemDiskName = blkDev.Name
			}
		}
		disks[blkDev.Name] = blkDev
	}
	instanceInfo.SystemDisk = types.DiskSlice{disks[systemDiskName]}
	delete(disks, systemDiskName)
	for _, value := range disks {
		if value.Type == "disk" {
			instanceInfo.DataDisk = append(instanceInfo.DataDisk, value)
		}
	}

	return &instanceInfo, nil
}

func parseNetCard(netDevStr string) types.NetWorkCard {
	netInfoMap := make(map[string]string)
	for _, pairStr := range strings.Split(netDevStr, " ") {
		pair := strings.Split(pairStr, "=")
		if len(pair) != 2 {
			logger.Error("parse net device pair: %s failed", pair)
			continue
		}
		value := pair[1]
		if value != "" {
			if value[0] == '"' {
				value = value[1:]
			}
			if value[len(value)-1] == '"' {
				value = value[:len(value)-1]
			}
		}
		netInfoMap[pair[0]] = value
	}
	return types.NetWorkCard{
		Name: netInfoMap["NAME"],
		IP:   netInfoMap["IP"],
		MAC:  netInfoMap["MAC"],
	}
}

func parseBlkDev(blkDevStr string) (*types.Disk, string, error) {
	blkInfoMap := make(map[string]string)
	for _, pairStr := range strings.Split(blkDevStr, " ") {
		pair := strings.Split(pairStr, "=")
		if len(pair) != 2 {
			return &types.Disk{}, "", fmt.Errorf("parse block device pair: %s failed", pair)
		}
		value := pair[1]
		if value != "" {
			if value[0] == '"' {
				value = value[1:]
			}
			if value[len(value)-1] == '"' {
				value = value[:len(value)-1]
			}
		}
		blkInfoMap[pair[0]] = value
	}

	capacityBit, err := strconv.Atoi(blkInfoMap["SIZE"])
	if err != nil {
		return &types.Disk{}, "", fmt.Errorf("block size: %s can't be convert to int", blkInfoMap["SIZE"])
	}

	return &types.Disk{
		Name:       blkInfoMap["NAME"],
		MountPoint: blkInfoMap["MOUNTPOINT"],
		FSType:     blkInfoMap["FSTYPE"],
		Capacity:   int32(capacityBit / (1024 * 1024 * 1024)),
		Type:       blkInfoMap["TYPE"],
	}, blkInfoMap["PKNAME"], nil
}

func getBetweenStr(str, begin, end string) (string, error) {
	n := strings.Index(str, begin)
	if n == -1 {
		return "", fmt.Errorf("can't find begin str")
	}

	m := strings.Index(str, end)
	if m == -1 {
		return "", fmt.Errorf("can't find end str")
	}

	return str[n+len(begin) : m], nil
}

func isSystemDisk(blkdev *types.Disk) bool {
	return blkdev.MountPoint == "/"
}

func IPFormat(host string) string {
	if !strings.ContainsRune(host, ':') {
		return host
	}
	return strings.Split(host, ":")[0]
}
