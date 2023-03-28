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

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/yaml"
)

func (k *Runtime) joinNodes(newNodes, masters []net.IP, kubeadmConfig kubeadm.KubeadmConfig, token v1beta3.BootstrapTokenDiscovery) error {
	if len(newNodes) == 0 {
		return nil
	}

	//TODO: bugfix: keep the same CRISocket with InitConfiguration
	if err := k.initKube(newNodes); err != nil {
		return err
	}

	kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken = &token
	kubeadmConfig.JoinConfiguration.Discovery.BootstrapToken.APIServerEndpoint = net.JoinHostPort(k.getAPIServerVIP().String(), "6443")
	kubeadmConfig.JoinConfiguration.ControlPlane = nil

	if err := k.configureLvs(masters, newNodes); err != nil {
		return fmt.Errorf("failed to configure lvs rule for apiserver: %v", err)
	}

	eg, _ := errgroup.WithContext(context.Background())

	for _, n := range newNodes {
		node := n
		eg.Go(func() error {
			logrus.Infof("start to join %s as worker", node)
			joinNodeCmd, err := k.Command(JoinNode, k.getNodeNameOverride(node))
			if err != nil {
				return err
			}

			myKubeadmConfig := kubeadmConfig

			if output, err := k.infra.CmdToString(node, nil, GetCustomizeCRISocket, ""); err == nil && output != "" {
				myKubeadmConfig.JoinConfiguration.NodeRegistration.CRISocket = output
			}
			joinConfig, err := yaml.MarshalWithDelimiter(myKubeadmConfig.JoinConfiguration, myKubeadmConfig.KubeletConfiguration)
			if err != nil {
				return err
			}
			writeJoinConfigCmd := fmt.Sprintf("mkdir -p /etc/kubernetes && echo \"%s\" > %s", joinConfig, KubeadmFileYml)

			err = k.checkMultiNetworkAddVIPRoute(node)
			if err != nil {
				return fmt.Errorf("failed to check multi network: %v", err)
			}

			if err = k.infra.CmdAsync(node, nil, writeJoinConfigCmd); err != nil {
				return fmt.Errorf("failed to set join kubeadm config on host(%s) with cmd(%s): %v", node, writeJoinConfigCmd, err)
			}

			if err = k.infra.CmdAsync(node, nil, joinNodeCmd); err != nil {
				return fmt.Errorf("failed to join node %s: %v", node, err)
			}

			logrus.Infof("succeeded in joining %s as worker", node)
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) checkMultiNetworkAddVIPRoute(node net.IP) error {
	result, err := k.infra.CmdToString(node, nil, fmt.Sprintf(RemoteCheckRoute, node), "")
	if err != nil {
		return err
	}
	if result == utilsnet.RouteOK {
		return nil
	}

	cmd := fmt.Sprintf(RemoteAddRoute, k.getAPIServerVIP(), node)
	if _, err := k.infra.Cmd(node, nil, cmd); err != nil {
		return err
	}
	return nil
}
