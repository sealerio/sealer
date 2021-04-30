// alibaba-inc.com Inc.
// Copyright (c) 2004-2021 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2021/3/15 9:03 上午
// @File : const
//

package runtime

const (
	ClusterRootfsWorkspace = "/var/lib/seadent/data/%s"
	WriteKubeadmConfigCmd  = "cd %s && echo \"%s\" > kubeadm-config.yaml"
	CreateKubeConfigCmd    = "mkdir -p ~/.kube && cp /etc/kubernetes/admin.conf ~/.kube/config"
	//CreateEtcdSecretCmd    = `kubectl create secret generic etcd-client-cert --from-file=ca.pem=/etc/kubernetes/pki/etcd/ca.crt --from-file=etcd-client.pem=/etc/kubernetes/pki/apiserver-etcd-client.crt --from-file=etcd-client-key.pem=/etc/kubernetes/pki/apiserver-etcd-client.key -n kube-system`
)
