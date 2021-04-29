package utils

import (
	"bytes"
	"net"
	"sort"
)

func NotIn(key string, slice []string) bool {
	for _, s := range slice {
		if key == s {
			return false
		}
	}
	return true
}

func ReduceIPList(src, dst []string) []string {
	var ipList []string
	for _, ip := range src {
		if !NotIn(ip, dst) {
			ipList = append(ipList, ip)
		}
	}
	return ipList
}

func AppendIPList(src, dst []string) []string {
	for _, ip := range dst {
		if NotIn(ip, src) {
			src = append(src, ip)
		}
	}
	return src
}

func SortIPList(iplist []string) {
	realIPs := make([]net.IP, 0, len(iplist))
	for _, ip := range iplist {
		realIPs = append(realIPs, net.ParseIP(ip))
	}

	sort.Slice(realIPs, func(i, j int) bool {
		return bytes.Compare(realIPs[i], realIPs[j]) < 0
	})

	for i, _ := range realIPs {
		iplist[i] = realIPs[i].String()
	}

}
