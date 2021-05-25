package utils

import (
	"encoding/hex"
	"net"
)

/*
	https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing
	CIDR表示法:
	IPv4   	网络号/前缀长度		192.168.1.0/24
	IPv6	接口号/前缀长度		2001:db8::/64
*/
type CIDR struct {
	ip    net.IP
	ipnet *net.IPNet
}

// 解析CIDR网段
func ParseCIDR(s string) (*CIDR, error) {
	i, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	return &CIDR{ip: i, ipnet: n}, nil
}

// 解析并校准CIDR
func ParseCIDRString(s string) (string, error) {
	c, err := ParseCIDR(s)
	if err != nil {
		return "", err
	}
	return c.CIDR(), nil
}

// 判断是否IPv4
func (c CIDR) IsIPv4() bool {
	_, bits := c.ipnet.Mask.Size()
	return bits/8 == net.IPv4len
}

// 判断是否IPv6
func (c CIDR) IsIPv6() bool {
	_, bits := c.ipnet.Mask.Size()
	return bits/8 == net.IPv6len
}

// CIDR字符串中的IP部分
func (c CIDR) IP() string {
	return c.ip.String()
}

// 网络号
func (c CIDR) Network() string {
	return c.ipnet.IP.String()
}

// 子网掩码位数
func (c CIDR) MaskSize() (ones, bits int) {
	ones, bits = c.ipnet.Mask.Size()
	return
}

// 子网掩码
func (c CIDR) Mask() string {
	mask, _ := hex.DecodeString(c.ipnet.Mask.String())
	return net.IP([]byte(mask)).String()
}

// 根据子网掩码长度校准后的CIDR
func (c CIDR) CIDR() string {
	return c.ipnet.String()
}
