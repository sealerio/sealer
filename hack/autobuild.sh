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

# -----------------------------------------------------------------------------
#The script format is similar to that of auto build：
#If we need to make kubernetes:1.20.14 image, We just need to enter: kubernetes v1.20.14 amd64
echo "Start the way ,eg: sh autobuild.sh v1.20.14 amd64"
version=$1
echo "version: $version"
arch=$2
echo "arch: $arch"
wget http://sealer.oss-cn-beijing.aliyuncs.com/auto-build/rootfs.tar.gz
tar -xvf rootfs.tar.gz
wget https://dl.k8s.io/$version/kubernetes-server-linux-$arch.tar.gz
tar -xvf kubernetes-server-linux-$arch.tar.gz
sudo cp ./kubernetes/server/bin/kubectl ./rootfs/bin/
sudo cp ./kubernetes/server/bin/kubeadm ./rootfs/bin/
sudo cp ./kubernetes/server/bin/kubelet ./rootfs/bin/
wget https://dl.k8s.io/$version/kubernetes-server-linux-amd64.tar.gz
tar -xvf kubernetes-server-linux-amd64.tar.gz
wget http://sealer.oss-cn-beijing.aliyuncs.com/auto-build/sealer.tar.gz
sudo tar -xvf sealer.tar.gz -C /usr/bin
sudo sed -i "s/v1.20.14/$version/g" ./rootfs/etc/kubeadm.yml
sudo sed -i "s/v1.20.14/$version/g" ./rootfs/Metadata
sudo sed -i "s/amd64/$arch/g" ./rootfs/Metadata
sudo ./kubernetes/server/bin/kubeadm config images list --config ./rootfs/etc/kubeadm.yml 2>/dev/null>>./rootfs/imageList
cd ./rootfs
sudo sealer build -f Kubefile -m lite -t kubernetes:$version .