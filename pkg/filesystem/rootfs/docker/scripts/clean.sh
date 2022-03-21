#!/bin/bash
# Copyright © 2021 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

systemctl stop docker
rm -rf /lib/systemd/system/docker.service
rm -rf /usr/lib/systemd/system/docker.service
rm -rf /etc/docker/daemon.json
systemctl daemon-reload

rm -f /usr/bin/conntrack
rm -f /usr/bin/kubelet-pre-start.sh
rm -f /usr/bin/containerd
rm -f /usr/bin/containerd-shim
rm -f /usr/bin/containerd-shim-runc-v2
rm -f /usr/bin/crictl
rm -f /usr/bin/ctr
rm -f /usr/bin/docker
rm -f /usr/bin/docker-init
rm -f /usr/bin/docker-proxy
rm -f /usr/bin/dockerd
rm -f /usr/bin/kubeadm
rm -f /usr/bin/kubetcl
rm -f /usr/bin/kubelet
rm -f /usr/bin/rootlesskit
rm -f /usr/bin/rootlesskit-docker-proxy
rm -f /usr/bin/runc
rm -f /usr/bin/vpnkit
rm -f /usr/bin/containerd-rootless-setuptool.sh
rm -f /usr/bin/containerd-rootless.sh
rm -f /usr/bin/nerdctl

rm -f /etc/sysctl.d/k8s.conf
rm -f /etc/systemd/system/kubelet.service
rm -rf /etc/systemd/system/kubelet.service.d
rm -rf /var/lib/kubelet/
rm -f /var/lib/kubelet/config.yaml
