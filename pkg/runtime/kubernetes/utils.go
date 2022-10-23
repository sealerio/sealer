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

package kubernetes

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	versionUtils "github.com/sealerio/sealer/utils/version"
)

const (
	AuditPolicyYml = "audit-policy.yml"
	KubeadmFileYml = "/etc/kubernetes/kubeadm.yaml"
	AdminConf      = "admin.conf"
	ControllerConf = "controller-manager.conf"
	SchedulerConf  = "scheduler.conf"
	KubeletConf    = "kubelet.conf"

	// kube file
	KUBECONTROLLERCONFIGFILE = "/etc/kubernetes/controller-manager.conf"
	KUBESCHEDULERCONFIGFILE  = "/etc/kubernetes/scheduler.conf"
	AdminKubeConfPath        = "/etc/kubernetes/admin.conf"
	StaticPodDir             = "/etc/kubernetes/manifests"
	LvscarePodFileName       = "kube-lvscare.yaml"
)

const (
	RemoteAddEtcHosts       = "cat /etc/hosts |grep '%s' || echo '%s' >> /etc/hosts"
	RemoteReplaceKubeConfig = `grep -qF "apiserver.cluster.local" %s  && sed -i 's/apiserver.cluster.local/%s/' %s && sed -i 's/apiserver.cluster.local/%s/' %s`
	RemoveKubeConfig        = "rm -rf /usr/bin/kube* && rm -rf ~/.kube/"
	RemoteCleanK8sOnHost    = `if which kubeadm > /dev/null 2>&1;then kubeadm reset -f %s;fi && \
rm -rf /etc/kubernetes/ && \
rm -rf /etc/systemd/system/kubelet.service.d && rm -rf /etc/systemd/system/kubelet.service && \
rm -rf /usr/bin/kubeadm && rm -rf /usr/bin/kubelet-pre-start.sh && \
rm -rf /usr/bin/kubelet && rm -rf /usr/bin/kubectl && \
rm -rf /var/lib/kubelet/* && rm -rf /etc/sysctl.d/k8s.conf && \
rm -rf /etc/cni && rm -rf /opt/cni && \
rm -rf /var/lib/etcd && rm -rf /var/etcd
`
	RemoteRemoveAPIServerEtcHost = "sed -i \"/%s/d\" /etc/hosts"
	RemoveLvscareStaticPod       = "rm -rf  /etc/kubernetes/manifests/kube-sealyun-lvscare*"
	KubeDeleteNode               = "kubectl delete node %s"

	CreateLvscareStaticPod = "mkdir -p %s && echo \"%s\" > %s"
	RemoteCheckRoute       = "seautil route check --host %s"
	RemoteAddRoute         = "seautil route add --host %s --gateway %s"
	RemoteDelRoute         = "if command -v seautil > /dev/null 2>&1; then seautil route del --host %s --gateway %s; fi"
)

// StaticFile :static file should not be template, will never be changed while initialization.
type StaticFile struct {
	DestinationDir string
	Name           string
}

// MasterStaticFiles Put static files here, can be moved to all master nodes before kubeadm execution
var MasterStaticFiles = []*StaticFile{
	{
		DestinationDir: "/etc/kubernetes",
		Name:           AuditPolicyYml,
	},
}

// return node name from k8s cluster, if not found, return "" and error is nil
func (k *Runtime) getNodeNameByCmd(master, host net.IP) (string, error) {
	//todo get node name from k8s sdk
	cmd := fmt.Sprintf("kubectl get nodes -o wide | grep -v NAME  | grep %s | awk '{print $1}'", host)
	hostName, err := k.infra.CmdToString(master, cmd, "")
	if err != nil {
		return "", err
	}

	hostString, err := k.infra.CmdToString(master, "kubectl get nodes | grep -v NAME  | awk '{print $1}'", ",")
	if err != nil {
		return "", err
	}
	nodeNames := strings.Split(hostString, ",")

	for _, nodeName := range nodeNames {
		if strings.TrimSpace(nodeName) == "" {
			continue
		} else if strings.EqualFold(nodeName, hostName) {
			return nodeName, nil
		}
	}

	return "", fmt.Errorf("failed to find node name form %s", host.String())
}

func vlogToStr(vlog int) string {
	str := strconv.Itoa(vlog)
	return " -v " + str
}

type CommandType string

const InitMaster CommandType = "initMaster"
const JoinMaster CommandType = "joinMaster"
const JoinNode CommandType = "joinNode"

func (k *Runtime) Command(version, master0IP string, name CommandType, token v1beta2.BootstrapTokenDiscovery, certKey string) (string, error) {
	//cmds := make(map[CommandType]string)
	// Please convert your v1beta1 configuration files to v1beta2 using the
	// "kubeadm config migrate" command of kubeadm v1.15.x, so v1.14 not support multi network interface.
	cmds := map[CommandType]string{
		InitMaster: fmt.Sprintf("kubeadm init --config=%s/etc/kubeadm.yml --experimental-upload-certs", k.infra.GetClusterRootfsPath()),
		JoinMaster: fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s --experimental-control-plane --certificate-key %s", net.JoinHostPort(master0IP, "6443"), token.Token, token.CACertHashes, certKey),
		JoinNode:   fmt.Sprintf("kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s", net.JoinHostPort(k.getAPIServerVIP().String(), "6443"), token.Token, token.CACertHashes),
	}

	kv := versionUtils.Version(version)
	cmp, err := kv.Compare(kubeadm.V1150)
	//other version >= 1.15.x
	if err != nil {
		logrus.Errorf("failed to compare Kubernetes version: %s", err)
	}
	if cmp {
		cmds[InitMaster] = fmt.Sprintf("kubeadm init --config=%s --upload-certs", KubeadmFileYml)
		cmds[JoinMaster] = fmt.Sprintf("kubeadm join --config=%s", KubeadmFileYml)
		cmds[JoinNode] = fmt.Sprintf("kubeadm join --config=%s", KubeadmFileYml)
	}

	v, ok := cmds[name]
	if !ok {
		return "", fmt.Errorf("failed to get kubeadm command: %v", cmds)
	}

	if runtime.IsInContainer() {
		return fmt.Sprintf("%s%s%s", v, vlogToStr(k.Config.Vlog), " --ignore-preflight-errors=all"), nil
	}
	if name == InitMaster || name == JoinMaster {
		return fmt.Sprintf("%s%s%s", v, vlogToStr(k.Config.Vlog), " --ignore-preflight-errors=SystemVerification"), nil
	}

	return fmt.Sprintf("%s%s", v, vlogToStr(k.Config.Vlog)), nil
}

func GetClientFromConfig(adminConfPath string) (runtimeClient.Client, error) {
	adminConfig, err := clientcmd.BuildConfigFromFlags("", adminConfPath)
	if nil != err {
		return nil, err
	}

	var ret runtimeClient.Client

	timeout := time.Second * 30
	err = wait.PollImmediate(time.Second*10, timeout, func() (done bool, err error) {
		cli, err := runtimeClient.New(adminConfig, runtimeClient.Options{})
		if nil != err {
			return false, err
		}

		ns := corev1.Namespace{}
		if err := cli.Get(context.Background(), runtimeClient.ObjectKey{Name: "default"}, &ns); nil != err {
			return false, err
		}

		ret = cli

		return true, nil
	})

	return ret, err
}
