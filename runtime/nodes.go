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
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/ipvs"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
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

func (d *Default) joinNodes(nodes []string) error {
	if len(nodes) == 0 {
		return nil
	}
	if err := d.LoadMetadata(); err != nil {
		return fmt.Errorf("failed to load metadata %v", err)
	}
	if err := ssh.WaitSSHReady(d.SSH, nodes...); err != nil {
		return errors.Wrap(err, "join nodes wait for ssh ready time out")
	}
	if err := d.GetJoinTokenHashAndKey(); err != nil {
		return err
	}
	var masters string
	var wg sync.WaitGroup
	for _, master := range d.Masters {
		masters += fmt.Sprintf(" --rs %s:6443", utils.GetHostIP(master))
	}
	ipvsCmd := fmt.Sprintf(RemoteAddIPVS, d.VIP, masters)
	templateData := string(d.JoinTemplate(""))
	cmdAddRegistryHosts := fmt.Sprintf(RemoteAddEtcHosts, getRegistryHost(utils.GetHostIP(d.Masters[0])))
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			/*
				cmdRoute := fmt.Sprintf(RemoteCheckRoute, utils.GetHostIP(node))
				status := d.CmdToString(node, cmdRoute, "")
				if status != RouteOK {
					addRouteCmd := fmt.Sprintf(RemoteAddRoute, d.VIP, utils.GetHostIP(node))
					d.CmdToString(node, addRouteCmd, "")
				}
			*/
			// send join node config
			cmdJoinConfig := fmt.Sprintf(RemoteJoinConfig, templateData, d.Rootfs)
			cmdHosts := fmt.Sprintf(RemoteAddIPVSEtcHosts, d.VIP, d.APIServer)
			cmd := d.Command(d.Metadata.Version, JoinNode)
			yaml := ipvs.LvsStaticPodYaml(d.VIP, d.Masters, d.LvscareImage)
			lvscareStaticCmd := fmt.Sprintf(LvscareStaticPodCmd, yaml, LvscareDefaultStaticPodFileName)
			if err := d.SSH.CmdAsync(node, cmdAddRegistryHosts, cmdJoinConfig, cmdHosts, ipvsCmd, cmd, RemoteStaticPodMkdir, lvscareStaticCmd); err != nil {
				logger.Error("exec commands failed %s %v", node, err)
			}
		}(node)
	}

	wg.Wait()
	return nil
}

func (d *Default) deleteNodes(nodes []string) error {
	if len(nodes) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			if err := d.deleteNode(node); err != nil {
				logger.Error("delete node %s failed %v", node, err)
			}
		}(node)
	}
	wg.Wait()

	return nil
}

func (d *Default) deleteNode(node string) error {
	host := utils.GetHostIP(node)
	if err := d.SSH.CmdAsync(host, fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(d.Vlog)), fmt.Sprintf(RemoteRemoveAPIServerEtcHost, d.APIServer), fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(d.Masters[0]))); err != nil {
		return err
	}

	//remove node
	if len(d.Masters) > 0 {
		hostname := d.isHostName(d.Masters[0], node)
		err := d.SSH.CmdAsync(d.Masters[0], fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname)))
		if err != nil {
			return fmt.Errorf("delete node %s failed %v", hostname, err)
		}
	}

	return nil
}
