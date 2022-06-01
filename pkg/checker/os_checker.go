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

package checker

import (
	"fmt"
	"strings"

	"github.com/sealerio/sealer/pkg/clusterinfo"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// OS contains the OS name and the kernel version
type OS struct {
	OSName        string `json:"osName,omitempty" yaml:"osName,omitempty"`
	OSVersion     string `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	KernelVersion string `json:"kernelVersion,omitempty" yaml:"kernelVersion,omitempty"`
	Arch          string `json:"arch,omitempty" yaml:"arch,omitempty"`
}

func (sro *OS) GetStr() string {
	return fmt.Sprintf("{Arch:%s, OS:%s, Version:%s, Kernel:%s}", sro.Arch, sro.OSName, sro.OSVersion, sro.KernelVersion)
}

var SupportedOS = []OS{
	{OSName: "CentOS", OSVersion: "7.7", KernelVersion: "3.10.0", Arch: "amd64"},
	{OSName: "CentOS", OSVersion: "7.8", KernelVersion: "3.10.0", Arch: "amd64"},
	{OSName: "CentOS", OSVersion: "8.2", KernelVersion: "4.18", Arch: "amd64"},
	{OSName: "Ubuntu", OSVersion: "20.04", KernelVersion: "4.18", Arch: "amd64"},
	{OSName: "Ubuntu", OSVersion: "18.04", KernelVersion: "4.15", Arch: "amd64"},
}

func GetSupportedOSStr() []string {
	arr := make([]string, 0)
	for _, sro := range SupportedOS {
		arr = append(arr, fmt.Sprintf("{Arch:%s, OS:%s, Version:%s.*, Kernel:%s.*}", sro.Arch, sro.OSName, sro.OSVersion, sro.KernelVersion))
	}
	return arr
}

type OsChecker struct {
}

func NewOsChecker() Interface {
	return &OsChecker{}
}

func (o OsChecker) Check(cluster *v2.Cluster, phase string) error {
	detailed, err := clusterinfo.GetClusterInfo(cluster)
	if err != nil {
		return err
	}
	for _, instance := range detailed.InstanceInfos {
		os := OS{
			OSName:        instance.OS,
			OSVersion:     instance.OSVersion,
			KernelVersion: instance.Kernel,
			Arch:          instance.Arch,
		}
		match := false
		for _, r := range SupportedOS {
			if r.OSName == os.OSName &&
				strings.HasPrefix(os.OSVersion, r.OSVersion) &&
				strings.HasPrefix(os.KernelVersion, r.KernelVersion) &&
				r.Arch == os.Arch {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("the current host is: \n%s.\nthe OS only support: \n%s", os.GetStr(), strings.Join(GetSupportedOSStr(), ",\n"))
		}
	}
	return nil
}
