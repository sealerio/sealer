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

set -x
set -e
if ! [ -x /usr/local/bin/ctr ]; then
  tar  -xvzf ../cri/containerd.tar.gz -C /
  [ -f /usr/lib64/libseccomp.so.2 ] || cp -rf ../lib64/lib* /usr/lib64/
  systemctl enable  containerd.service
  systemctl restart containerd.service
fi

mkdir -p /etc/containerd

sed -i "s/sea.hub/${1:-sea.hub}/g" ../etc/dump-config.toml
sed -i "s/5000/${2:-5000}/g" ../etc/dump-config.toml

#add cri sandbox image and sea.hub registry cert path
##sandbox_image = "sea.hub:5000/pause:3.6" custom setup
containerd --config ../etc/dump-config.toml config dump > /etc/containerd/config.toml

systemctl restart containerd.service