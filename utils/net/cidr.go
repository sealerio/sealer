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
	"encoding/hex"
	"net"
)

/*
	https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing
	CIDR doc:
	IPv4   	network Addr/prefixLength		192.168.1.0/24
	IPv6	network Addr/prefixLength		2001:db8::/64
*/
type CIDR struct {
	ip    net.IP
	ipnet *net.IPNet
}

// ParseCIDR Parse CIDR net range
func ParseCIDR(s string) (*CIDR, error) {
	i, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	return &CIDR{ip: i, ipnet: n}, nil
}

// ParseCIDRString Parse CIDR  Fix to Standard CIDR
func ParseCIDRString(s string) (string, error) {
	c, err := ParseCIDR(s)
	if err != nil {
		return "", err
	}
	return c.CIDR(), nil
}

// IsIPv4 check it is IPv4
func (c CIDR) IsIPv4() bool {
	_, bits := c.ipnet.Mask.Size()
	return bits/8 == net.IPv4len
}

// IsIPv6 check it is IPv6
func (c CIDR) IsIPv6() bool {
	_, bits := c.ipnet.Mask.Size()
	return bits/8 == net.IPv6len
}

// IP Get ip address
func (c CIDR) IP() string {
	return c.ip.String()
}

// Network Get Network Addr
func (c CIDR) Network() string {
	return c.ipnet.IP.String()
}

// MaskSize Get mask Size
func (c CIDR) MaskSize() (ones, bits int) {
	ones, bits = c.ipnet.Mask.Size()
	return
}

// Mask get subset mask
func (c CIDR) Mask() string {
	mask, _ := hex.DecodeString(c.ipnet.Mask.String())
	return net.IP(mask).String()
}

// CIDR get fixed CIDR String
func (c CIDR) CIDR() string {
	return c.ipnet.String()
}
