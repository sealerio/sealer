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
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) reset(mastersToDelete, workersToDelete []net.IP) error {
	all := append(mastersToDelete, workersToDelete...)
	remoteCleanCmd := []string{fmt.Sprintf(RemoteCleanK8sOnHost, vlogToStr(k.Config.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, k.getAPIServerDomain())}

	// do kubeadm reset
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range all {
		n := node
		eg.Go(func() error {
			if err := k.infra.CmdAsync(n, nil, remoteCleanCmd...); err != nil {
				return err
			}
			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		return err
	}

	// clean vip route on node
	for _, node := range workersToDelete {
		if err = k.deleteVIPRouteIfExist(node); err != nil {
			return fmt.Errorf("failed to delete vip route %s: %v", node, err)
		}
	}
	return nil
}

func (k *Runtime) deleteMasters(mastersToDelete, remainMasters []net.IP) error {
	var remainMaster0 *net.IP
	if len(remainMasters) > 0 {
		remainMaster0 = &remainMasters[0]
	}

	eg, _ := errgroup.WithContext(context.Background())

	for _, m := range mastersToDelete {
		master := m
		eg.Go(func() error {
			logrus.Infof("start to delete master %s", master)
			if err := k.deleteMaster(master, remainMaster0); err != nil {
				return fmt.Errorf("failed to delete master %s: %v", master, err)
			}
			logrus.Infof("succeeded in deleting master %s", master)

			return nil
		})
	}

	return eg.Wait()
}

func (k *Runtime) deleteMaster(master net.IP, remainMaster0 *net.IP) error {
	remoteCleanCmd := []string{fmt.Sprintf(RemoteCleanK8sOnHost, vlogToStr(k.Config.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, k.getAPIServerDomain()), RemoteCleanCustomizeCRISocket}

	//if the master to be removed is the execution machine, kubelet and ~./kube will not be removed and ApiServer host will be added.

	if err := k.infra.CmdAsync(master, nil, remoteCleanCmd...); err != nil {
		logrus.Warnf("failed to run remote cleanup cmd on master %s, ignore and continue remove it from cluster", master)
	}

	// if remainMaster0 is nil, no need delete master from cluster
	if remainMaster0 != nil {
		hostname, err := k.getNodeNameByCmd(master)
		if err != nil {
			logrus.Infof("%v, just think it not in k8s and skip delete it", err)
			return nil
		}
		if err = k.infra.CmdAsync(*remainMaster0, nil, fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("failed to delete master %s: %v", hostname, err)
		}
	}

	return nil
}

func (k *Runtime) deleteNodes(nodesToDelete, remainMasters []net.IP) error {
	var remainMaster0 *net.IP
	if len(remainMasters) > 0 {
		remainMaster0 = &remainMasters[0]
	}

	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodesToDelete {
		n := node
		eg.Go(func() error {
			logrus.Infof("start to delete worker %s", n)
			if err := k.deleteNode(n, remainMaster0); err != nil {
				return fmt.Errorf("failed to delete node %s: %v", n, err)
			}
			logrus.Infof("succeeded in deleting worker %s", n)

			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) deleteNode(node net.IP, remainMaster0 *net.IP) error {
	remoteCleanCmd := []string{fmt.Sprintf(RemoteCleanK8sOnHost, vlogToStr(k.Config.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, k.getAPIServerDomain()), RemoteCleanCustomizeCRISocket, fmt.Sprintf(RemoteDelRoute, k.getAPIServerVIP(), node)}

	//if the master to be removed is the execution machine, kubelet and ~./kube will not be removed and ApiServer host will be added.

	if err := k.infra.CmdAsync(node, nil, remoteCleanCmd...); err != nil {
		logrus.Warnf("failed to run remote cleanup cmd on node %s, ignore and continue remove it from cluster", node)
	}

	// if remainMaster0 is nil, no need delete master from cluster
	if remainMaster0 != nil {
		hostname, err := k.getNodeNameByCmd(node)
		if err != nil {
			logrus.Infof("%v, just think it not in k8s and skip delete it", err)
			return nil
		}

		if err = k.infra.CmdAsync(*remainMaster0, nil, fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("failed to delete node %s: %v", hostname, err)
		}
	}

	return nil
}

func (k *Runtime) deleteVIPRouteIfExist(node net.IP) error {
	_, err := k.infra.Cmd(node, nil, fmt.Sprintf(RemoteDelRoute, k.getAPIServerVIP(), node))
	return err
}
