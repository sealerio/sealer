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

helpfunc() {
    echo "Usage:"
    echo "    serverstart.sh [-i HOST] [-d DIR_LIST]"
    echo "Description:"
    echo "    HOST: the IP of nydusdserver."
    echo "    DIR_LIST: the list of dir need to be converted to nydus blobs."
    echo "examples:"
    echo "    serverstart.sh -i 192.168.0.2 -d /converdir/1,/converdir/2,/converdir/2"
    exit -1
}

while getopts 'i:d:h' OPT; do
    case $OPT in
        i) HOST="$OPTARG";;
        d) DIR_LIST="$OPTARG";;
        h) helpfunc;;
        ?) helpfunc;;
    esac
done

nydusdconfig='
{\n
  "device": {\n
    "backend": {\n
      "type": "registry",\n
      "config": {\n
        "scheme": "http",\n
        "host": "'${HOST}':8000",\n
        "repo": "sealer"\n
      }\n
    },\n
    "cache": {\n
      "type": "blobcache",\n
      "config": {\n
        "work_dir": "./cache"\n
      }\n
    }\n
  },\n
  "mode": "direct",\n
  "digest_validate": false,\n
  "enable_xattr": true,\n
  "fs_prefetch": {\n
    "enable": true,\n
    "threads_count": 1,\n
    "merging_size": 131072,\n
    "bandwidth_rate":10485760\n
  }\n
}\n
'
echo -e ${nydusdconfig} > ./httpserver.json

for DIR in $(echo "${DIR_LIST}"|sed 's/,/ /g')
do
    # create nydusimages
    F_NAME=$(basename "${DIR}")
    echo $F_NAME
    mkdir -p ../${F_NAME}
    ./nydus-image create --blob-dir ./nydusblobs  --bootstrap ../${F_NAME}/rootfs.meta $DIR
    cp ./httpserver.json ../${F_NAME}
done

rm -rf /usr/bin/nydus-backend-proxy
cp -u nydus-backend-proxy /usr/bin/nydus-backend-proxy
# nydusd_http_server.service
service="[Unit]\n
Description=A simple HTTP server to serve a local directory as blob backend for nydusd\n
[Service]\n
TimeoutStartSec=3\n
Environment=\"ROCKET_CONFIG=$(pwd)/Rocket.toml\"\n
ExecStart=/usr/bin/nydus-backend-proxy --blobsdir $(pwd)/nydusblobs\n
Restart=always\n
[Install]\n
WantedBy=multi-user.target\n"
echo -e ${service} > /etc/systemd/system/nydusd_http_server.service
# start nydusd_http_server.service
systemctl enable nydusd_http_server.service
systemctl restart nydusd_http_server.service


