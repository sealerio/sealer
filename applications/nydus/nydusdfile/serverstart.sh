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
# create nydusimages
./nydus-image create --blob-dir ./nydusdimages/blobs  --bootstrap ./rootfs.meta $1
#nydusdir need be scp
rm -rf $2
mkdir $2
cp ./nydusd_scp_file/* $2
cp ./rootfs.meta $2
# nydusd_http_server.service
service="[Unit]\n
Description=sealer nydusd rootfs\n
[Service]\n
TimeoutStartSec=3\n
Environment=\"ROCKET_CONFIG=$(pwd)/Rocket.toml\"\n
ExecStart=$(pwd)/nydusdserver $(pwd)/nydusdimages\n
Restart=always\n
[Install]\n
WantedBy=multi-user.target"
echo -e ${service} > /etc/systemd/system/nydusd_http_server.service
# start nydusd_http_server.service
systemctl enable nydusd_http_server.service
systemctl restart nydusd_http_server.service


