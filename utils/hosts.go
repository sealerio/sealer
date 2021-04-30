package utils

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func GetDiffHosts(hostsOld, hostsNew v1.Hosts) (add, sub []string) {
	diffMap := make(map[string]bool)
	for _, v := range hostsOld.IPList {
		diffMap[v] = true
	}
	for _, v := range hostsNew.IPList {
		if !diffMap[v] {
			add = append(add, v)
		} else {
			diffMap[v] = false
		}
	}
	for _, v := range hostsOld.IPList {
		if diffMap[v] {
			sub = append(sub, v)
		}
	}

	return
}
