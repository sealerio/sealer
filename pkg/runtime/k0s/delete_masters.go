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

package k0s

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) deleteMasters(mastersToDelete, remainMasters []net.IP) error {
	var remainMaster0 *net.IP
	if len(remainMasters) > 0 {
		remainMaster0 = &remainMasters[0]
		logrus.Infof("Master changed, remain master0 is: %s", remainMaster0)
	}

	eg, _ := errgroup.WithContext(context.Background())
	for _, m := range mastersToDelete {
		master := m
		eg.Go(func() error {
			logrus.Infof("Start to delete master %s", master)
			if err := k.deleteMaster(master); err != nil {
				return fmt.Errorf("failed to delete master %s: %v", master, err)
			}
			logrus.Infof("Succeeded in deleting master %s", master)
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) deleteMaster(master net.IP) error {
	/** To delete a node from k0s cluster, following these steps.
	STEP1: stop k0s service
	STEP2: reset the node with install configuration
	STEP3: remove k0s cluster config generate by k0s under /etc/k0s
	STEP4: remove private registry config in /etc/host
	STEP5: remove bin file such as: kubectl, and remove .kube directory
	STEP6: remove k0s bin file
	STEP7: no need to delete node though k8s client, cause k0s don't show master node in k8s cluster
	*/
	remoteCleanCmd := []string{"k0s stop",
		"k0s reset",
		"rm -rf /etc/k0s/",
		"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
		"rm -rf /usr/bin/k0s"}
	if err := k.infra.CmdAsync(master, nil, remoteCleanCmd...); err != nil {
		return err
	}
	return nil
}
