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

package net

import (
	"bytes"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"

	k8snet "k8s.io/apimachinery/pkg/util/net"
)

func GetHostNetInterface(host net.IP) (string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) == 0 {
			continue
		}
		addrs, err := netInterfaces[i].Addrs()
		if err != nil {
			return "", fmt.Errorf("failed to get Addrs: %v", err)
		}
		if IsLocalIP(host, addrs) {
			return netInterfaces[i].Name, nil
		}
	}
	return "", nil
}

func GetLocalHostAddresses() ([]net.Addr, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get net.Interfaces: %v", err)
	}

	var allAddrs []net.Addr
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) == 0 {
			continue
		}
		addrs, err := netInterfaces[i].Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get Addrs: %v", err)
		}
		for j := 0; j < len(addrs); j++ {
			allAddrs = append(allAddrs, addrs[j])
		}
	}
	return allAddrs, nil
}

func IsLocalIP(ip net.IP, addrs []net.Addr) bool {
	if len(addrs) == 0 {
		addrs, _ = GetLocalHostAddresses()
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.Equal(ip) {
			return true
		}
	}
	return false
}

func GetLocalDefaultIP() (string, error) {
	netIP, err := k8snet.ChooseHostInterface()
	if err != nil {
		return "", fmt.Errorf("failed to get default route ip, err: %v", err)
	}
	return netIP.String(), nil
}

func GetLocalIP(master0IPPort string) (net.IP, error) {
	conn, err := net.Dial("udp", master0IPPort)
	if err != nil {
		return nil, err
	}
	localAddr := conn.LocalAddr().String()
	return net.ParseIP(strings.Split(localAddr, ":")[0]), err
}

// TransferToIPList transfer network segment string to ip list string
func TransferToIPList(ipStr string) (string, error) {
	var result []string
	var ips = strings.Split(ipStr, "-")
	if ipStr == "" || !strings.Contains(ipStr, "-") {
		return ipStr, nil
	}
	if len(ips) != 2 {
		return "", fmt.Errorf("input IP(%s) is invalid, IP range format must be xxx.xxx.xxx.1-xxx.xxx.xxx.2", ipStr)
	}
	if returnedIP := net.ParseIP(ips[0]); returnedIP == nil {
		return "", fmt.Errorf("failed tp parse IP(%s)", ips[0])
	}
	if returnedIP := net.ParseIP(ips[1]); returnedIP == nil {
		return "", fmt.Errorf("failed tp parse IP(%s)", ips[1])
	}

	//ips[0],ips[1] = 192.168.56.3, 192.168.56.7;  result = [192.168.56.3, 192.168.56.4, 192.168.56.5, 192.168.56.6, 192.168.56.7]
	for res := CompareIP(ips[0], ips[1]); res <= 0; {
		result = append(result, ips[0])
		ips[0] = NextIP(ips[0]).String()
		res = CompareIP(ips[0], ips[1])
	}
	if len(result) == 0 {
		return "", fmt.Errorf("input IP(%s) is invalid", ipStr)
	}
	return strings.Join(result, ","), nil
}

func IPToInt(v string) *big.Int {
	ip := net.ParseIP(v).To4()
	if val := ip.To4(); val != nil {
		return big.NewInt(0).SetBytes(val)
	}
	return big.NewInt(0).SetBytes(ip.To16())
}

func CompareIP(v1, v2 string) int {
	i := IPToInt(v1)
	j := IPToInt(v2)

	if i == nil {
		return 2
	}
	if j == nil {
		return 2
	}

	return i.Cmp(j)
}

func NextIP(ip string) net.IP {
	i := IPToInt(ip)
	return i.Add(i, big.NewInt(1)).Bytes()
}

func SortIPList(iplist []string) {
	realIPs := make([]net.IP, 0, len(iplist))
	for _, ip := range iplist {
		realIPs = append(realIPs, net.ParseIP(ip))
	}

	sort.Slice(realIPs, func(i, j int) bool {
		return bytes.Compare(realIPs[i], realIPs[j]) < 0
	})

	for i := range realIPs {
		iplist[i] = realIPs[i].String()
	}
}

func IsInIPList(key net.IP, slice []net.IP) bool {
	for _, s := range slice {
		if s.Equal(key) {
			return true
		}
	}
	return false
}

func IPStrsToIPs(ipStrs []string) []net.IP {
	if ipStrs == nil {
		return nil
	}

	var result []net.IP
	for _, ipStr := range ipStrs {
		if ipStr == "" {
			continue
		}
		result = append(result, net.ParseIP(ipStr))
	}
	return result
}

func IPsToIPStrs(ips []net.IP) []string {
	if ips == nil {
		return nil
	}

	var result []string
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result
}

func RemoveIPs(clusterIPList []net.IP, toBeDeletedIPList []net.IP) (res []net.IP) {
	for _, ip := range clusterIPList {
		if !IsInIPList(ip, toBeDeletedIPList) {
			res = append(res, ip)
		}
	}
	return
}

func GetIndexIP(subnet *net.IPNet, index int) (string, error) {
	bip := big.NewInt(0).SetBytes(subnet.IP.To4())
	if subnet.IP.To4() == nil {
		bip = big.NewInt(0).SetBytes(subnet.IP.To16())
	}
	ip := net.IP(big.NewInt(0).Add(bip, big.NewInt(int64(index))).Bytes())
	if subnet.Contains(ip) {
		return ip.String(), nil
	}

	return "", fmt.Errorf("can't generate IP with index %d from subnet. subnet too small. subnet: %q", index, subnet)
}
