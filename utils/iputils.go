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

package utils

import (
	"fmt"
	"net"
	"strings"
)

//use only one
func GetHostIP(host string) string {
	if !strings.ContainsRune(host, ':') {
		return host
	}
	return strings.Split(host, ":")[0]
}

func GetHostIPAndPortOrDefault(host, Default string) (string, string) {
	if !strings.ContainsRune(host, ':') {
		return host, Default
	}
	split := strings.Split(host, ":")
	return split[0], split[1]
}

func GetSSHHostIPAndPort(host string) (string, string) {
	return GetHostIPAndPortOrDefault(host, "22")
}

func GetHostIPSlice(hosts []string) (res []string) {
	for _, ip := range hosts {
		res = append(res, GetHostIP(ip))
	}
	return
}

func IsLocalHostAddrs() (*[]net.Addr, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
		return nil, err
	}
	var allAddrs []net.Addr
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) == 0 {
			continue
		}
		addrs, err := netInterfaces[i].Addrs()
		if err != nil {
			fmt.Printf("failed to get Addrs, %s", err.Error())
		}
		for j := 0; j < len(addrs); j++ {
			allAddrs = append(allAddrs, addrs[j])
		}
	}
	return &allAddrs, nil
}

func IsLocalIP(ip string, addrs *[]net.Addr) bool {
	for _, address := range *addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil && ipnet.IP.String() == ip {
			return true
		}
	}
	return false
}
