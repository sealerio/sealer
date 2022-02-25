#!/bin/bash
# Copyright © 2022 Alibaba Group Holding Ltd.
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

rm -rf $1
mkdir -p $1
rm -rf nydusdfs
mkdir -p nydusdfs
cp nydusd /usr/bin/nydusd
nydusdcmd="#!/bin/bash\n/usr/bin/nydusd --thread-num 10 --log-level debug --mountpoint $(pwd)/nydusdfs --apisock $(pwd)/nydusd.sock --id sealer --bootstrap $(pwd)/rootfs.meta --config $(pwd)/httpserver.json --supervisor $(pwd)/supervisor.sock"
echo -e ${nydusdcmd} > /var/lib/sealer/nydusd.sh
chmod +x /var/lib/sealer/nydusd.sh
cp nydusd.service /etc/systemd/system/
systemctl enable nydusd.service
systemctl restart nydusd.service
rm -rf upper
rm -rf work
mkdir -p upper
mkdir -p work
sleep 0.5
mount -t overlay overlay -o lowerdir=$(pwd)/nydusdfs,upperdir=$(pwd)/upper,workdir=$(pwd)/work $1 -o index=off
