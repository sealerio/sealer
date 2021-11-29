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

package runtime

import (
	"fmt"
	"github.com/alibaba/sealer/utils"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/ipvs"
)

const (
	RemoteAddIPVS                   = "seautil ipvs --vs %s:6443 %s --health-path /healthz --health-schem https --run-once"
	RemoteStaticPodMkdir            = "mkdir -p /etc/kubernetes/manifests"
	RemoteJoinConfig                = `echo "%s" > %s/kubeadm-join-config.yaml`
	LvscareDefaultStaticPodFileName = "/etc/kubernetes/manifests/kube-lvscare.yaml"
	RemoteAddIPVSEtcHosts           = "echo %s %s >> /etc/hosts"
	RemoteCheckRoute                = "seautil route --host %s"
	RemoteAddRoute                  = "seautil route add --host %s --gateway %s"
	RouteOK                         = "ok"
	LvscareStaticPodCmd             = `echo "%s" > %s`
)

func (k *KubeadmRuntime) joinNodeConfig(nodeIp string) ([]byte, error) {
	// TODO get join config from config file
	k.setCgroupDriver(k.getCgroupDriverFromShell(nodeIp))
	k.setAPIServerEndpoint(fmt.Sprintf("%s:6443", k.getVIP()))
	k.setJoinLocalAPIEndpoint("", 0)
	return utils.MarshalConfigsYaml(k.JoinConfiguration, k.KubeletConfiguration)
}

func (k *KubeadmRuntime) joinNodes(nodes []string) error {
	errCh := make(chan error, len(nodes))
	defer close(errCh)

	if len(nodes) == 0 {
		return nil
	}
	if err := k.WaitSSHReady(6, nodes...); err != nil {
		return errors.Wrap(err, "join nodes wait for ssh ready time out")
	}
	if err := k.GetJoinTokenHashAndKey(); err != nil {
		return err
	}
	var masters string
	var wg sync.WaitGroup
	for _, master := range k.getMasterIPList() {
		masters += fmt.Sprintf(" --rs %s:6443", master)
	}
	ipvsCmd := fmt.Sprintf(RemoteAddIPVS, k.getVIP(), masters)

	k.setAPIServerEndpoint(fmt.Sprintf("%s:6443", k.getVIP()))
	k.JoinConfiguration.ControlPlane = nil

	cmdAddRegistryHosts := fmt.Sprintf(RemoteAddEtcHosts, getRegistryHost(k.getRootfs(), k.getMaster0IP()))
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			// send join node config, get cgroup driver on every join nodes
			joinConfig, err := k.joinNodeConfig(node)
			if err != nil {
				errCh <- fmt.Errorf("failed to join node %s %v", node, err)
				return
			}
			cmdJoinConfig := fmt.Sprintf(RemoteJoinConfig, string(joinConfig), k.getRootfs())
			cmdHosts := fmt.Sprintf(RemoteAddIPVSEtcHosts, k.getVIP(), k.getAPIServerDomain())
			cmd := k.Command(k.getKubeVersion(), JoinNode)
			yaml := ipvs.LvsStaticPodYaml(k.getVIP(), k.getMasterIPList(), "")
			lvscareStaticCmd := fmt.Sprintf(LvscareStaticPodCmd, yaml, LvscareDefaultStaticPodFileName)
			ssh, err := k.getHostSSHClient(node)
			if err != nil {
				errCh <- fmt.Errorf("failed to join node %s %v", node, err)
				return
			}
			if err := ssh.CmdAsync(node, cmdAddRegistryHosts, cmdJoinConfig, cmdHosts, ipvsCmd, cmd, RemoteStaticPodMkdir, lvscareStaticCmd); err != nil {
				errCh <- fmt.Errorf("failed to join node %s %v", node, err)
			}
		}(node)
	}

	wg.Wait()
	return ReadChanError(errCh)
}

func (k *KubeadmRuntime) deleteNodes(nodes []string) error {
	errCh := make(chan error, len(nodes))
	defer close(errCh)

	if len(nodes) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			if err := k.deleteNode(node); err != nil {
				errCh <- fmt.Errorf("delete node %s failed %v", node, err)
			}
		}(node)
	}
	wg.Wait()

	return ReadChanError(errCh)
}

func (k *KubeadmRuntime) deleteNode(node string) error {
	ssh, err := k.getHostSSHClient(node)
	if err != nil {
		return fmt.Errorf("failed to delete node: %v", err)
	}

	if err := ssh.CmdAsync(node, fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(k.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, k.getAPIServerDomain()),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(k.getRootfs(), k.getMaster0IP()))); err != nil {
		return err
	}

	//remove node
	if len(k.getMasterIPList()) > 0 {
		hostname := k.isHostName(k.getMaster0IP(), node)
		ssh, err := k.getHostSSHClient(k.getMaster0IP())
		if err != nil {
			return fmt.Errorf("failed to delete node on master0,%v", err)
		}
		if err := ssh.CmdAsync(k.getMaster0IP(), fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("delete node %s failed %v", hostname, err)
		}
	}

	return nil
}
