// alibaba-inc.com Inc.
// Copyright (c) 2004-2021 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2021/3/25 3:56 下午
// @File : static_files
//

package runtime

const (
	AuditPolicyYml = "audit-policy.yml"
)

// static file should not be template, will never be changed while initialization
type StaticFile struct {
	DestinationDir string
	Name           string
}

//MasterStaticFiles Put static files here, can be moved to all master nodes before kubeadm execution
var MasterStaticFiles = []*StaticFile{
	{
		DestinationDir: "/etc/kubernetes",
		Name:           AuditPolicyYml,
	},
}
