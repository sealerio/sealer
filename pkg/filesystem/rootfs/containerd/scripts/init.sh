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

set -e
set -x

#STORAGE=${1:-/var/lib/docker} compatible docker
REGISTRY_DOMAIN=${2-sea.hub}
REGISTRY_PORT=${3-5000}

# Install containerd
chmod a+x containerd.sh
sh containerd.sh "$REGISTRY_DOMAIN" "$REGISTRY_PORT"

# Modify kubelet conf
mkdir -p /etc/systemd/system/kubelet.service.d

if grep "SystemdCgroup = true" /etc/containerd/config.toml &>/dev/null; then
  driver=systemd
else
  driver=cgroupfs
fi

cat >/etc/systemd/system/kubelet.service.d/containerd.conf <<eof
[Service]
Environment="KUBELET_EXTRA_ARGS=--container-runtime=remote --cgroup-driver=${driver} --runtime-request-timeout=15m --container-runtime-endpoint=unix:///run/containerd/containerd.sock --image-service-endpoint=unix:///run/containerd/containerd.sock"
eof

chmod a+x init-kube.sh
sh init-kube.sh