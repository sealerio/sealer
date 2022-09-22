// Copyright © 2022 Alibaba Group Holding Ltd.
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
	"path"
	"strings"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config/v1beta2"
	"github.com/sealerio/sealer/utils/shellcommand"

	"github.com/sealerio/sealer/pkg/ipvs"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/yaml"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) joinNodes(newNodes, masters []net.IP, kubeadmConfig kubeadmconfig.KubeadmConfig, token v1beta2.BootstrapTokenDiscovery) error {
	if len(newNodes) == 0 {
		return nil
	}

	//TODO: bugfix: keep the same CRISocket with InitConfiguration
	if err := k.initKube(newNodes); err != nil {
		return err
	}

	var rs []string
	for _, m := range masters {
		rs = append(rs, fmt.Sprintf("--rs %s", net.JoinHostPort(m.String(), "6443")))
	}
	//set cluster VIP as APIServerEndpoint when join node
	vs := net.JoinHostPort(k.getAPIServerVIP().String(), "6443")
	ipvsCmd := fmt.Sprintf("seautil ipvs --vs %s %s --health-path /healthz --health-schem https --run-once", vs, strings.Join(rs, " "))

	kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken = &token
	kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken.APIServerEndpoint = vs
	kubeadmConfig.JoinConfiguration.ControlPlane = nil
	joinConfig, err := yaml.MarshalWithDelimiter(kubeadmConfig.JoinConfiguration, kubeadmConfig.KubeletConfiguration)
	if err != nil {
		return err
	}
	writeJoinConfigCmd := fmt.Sprintf("mkdir -p /etc/kubernetes && echo \"%s\" > %s", joinConfig, KubeadmFileYml)

	y := ipvs.LvsStaticPodYaml(k.getAPIServerVIP(), masters, k.Config.LvsImage)
	lvscareStaticCmd := fmt.Sprintf(CreateLvscareStaticPod, StaticPodDir, y, path.Join(StaticPodDir, LvscarePodFileName))

	joinNodeCmd, err := k.Command(kubeadmConfig.KubernetesVersion, masters[0].String(), JoinNode, token, "")
	if err != nil {
		return err
	}

	eg, _ := errgroup.WithContext(context.Background())

	for _, n := range newNodes {
		node := n
		eg.Go(func() error {
			logrus.Infof("Start to join %s as worker", node)

			err = k.checkMultiNetworkAddVIPRoute(node)
			if err != nil {
				return fmt.Errorf("failed to check multi network: %v", err)
			}

			if err = k.infra.CmdAsync(node, ipvsCmd); err != nil {
				return fmt.Errorf("failed to join node %s: %v", node, err)
			}

			if err = k.infra.CmdAsync(node, writeJoinConfigCmd); err != nil {
				return fmt.Errorf("failed to set join kubeadm config on host(%s) with cmd(%s): %v", node, writeJoinConfigCmd, err)
			}

			if err = k.infra.CmdAsync(node, shellcommand.CommandSetHostAlias(k.getAPIServerDomain(), k.getAPIServerVIP().String())); err != nil {
				return fmt.Errorf("failed to config cluster hosts file cmd: %v", err)
			}

			if err = k.infra.CmdAsync(node, joinNodeCmd); err != nil {
				return fmt.Errorf("failed to join node %s: %v", node, err)
			}

			if err = k.infra.CmdAsync(node, lvscareStaticCmd); err != nil {
				return fmt.Errorf("failed to set lvscare static pod %s: %v", node, err)
			}

			logrus.Infof("Succeeded in joining %s as worker", node)
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) checkMultiNetworkAddVIPRoute(node net.IP) error {
	result, err := k.infra.CmdToString(node, fmt.Sprintf(RemoteCheckRoute, node), "")
	if err != nil {
		return err
	}
	if result == utilsnet.RouteOK {
		return nil
	}
	_, err = k.infra.Cmd(node, fmt.Sprintf(RemoteAddRoute, k.getAPIServerVIP(), node))

	return err
}
