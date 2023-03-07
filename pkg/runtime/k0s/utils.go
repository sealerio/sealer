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
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"
)

const WaitingFork0sServiceStartTimes = 5

func (k *Runtime) WaitK0sReady(host net.IP) error {
	times := WaitingFork0sServiceStartTimes
	for {
		times--
		if times == 0 {
			break
		}
		logrus.Infof("waiting for k0s service ready")
		time.Sleep(time.Second * 5)
		bytes, err := k.infra.Cmd(host, nil, "k0s status")
		if err != nil {
			return err
		}
		// k0s status return: `Process ID: xxx` when it started successfully, or return: `connect failed`,
		// so we use field `Process` whether contains in string(bytes) to verify if k0s service started successfully.
		if strings.Contains(string(bytes), "Process") {
			return nil
		}
	}
	return errors.New("failed to start k0s: failed to get k0s status after 10 seconds")
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

func (k *Runtime) WaitSSHReady(tryTimes int, hosts ...net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			// TODO: use Time.Ticker to replace this loop.
			for i := 0; i < tryTimes; i++ {
				if err := k.infra.Ping(host); err == nil {
					return nil
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			return fmt.Errorf("wait for [%s] ssh ready timeout, ensure that the IP address or password is correct", host)
		})
	}
	return eg.Wait()
}

func (k *Runtime) getNodeName(host net.IP) (string, error) {
	client, err := k8s.NewK8sClient()
	if err != nil {
		return "", err
	}
	nodeList, err := client.ListNodes()
	if err != nil {
		return "", err
	}
	var hosts []string
	for _, node := range nodeList.Items {
		hosts = append(hosts, node.GetName())
	}

	hostName, err := k.infra.CmdToString(host, nil, "hostname", "")
	if err != nil {
		return "", err
	}

	var name string
	for _, h := range hosts {
		if strings.TrimSpace(h) == "" {
			continue
		}
		hh := strings.ToLower(h)
		fromH := strings.ToLower(hostName)
		if hh == fromH {
			name = h
			break
		}
	}
	return name, nil
}
