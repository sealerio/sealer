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
# prepare registry storage as directory
cd $(dirname $0)

REGISTRY_PORT=${1-5000}
VOLUME=${2-/var/lib/registry}

container=sealer-registry
rootfs=$(dirname "$(pwd)")
config="$rootfs/etc/registry_config.yml"
htpasswd="$rootfs/etc/registry_htpasswd"
certs_dir="$rootfs/certs"
image_dir="$rootfs/images"

mkdir -p $VOLUME || true

startRegistry() {
    n=1
    while (( $n <= 3 ))
    do
        echo "attempt to start registry"
        (docker start $container && break) || (( n < 3))
        (( n++ ))
        sleep 3
    done
}


# shellcheck disable=SC2045
for image in $(ls "$image_dir")
do
 if [ -f "$image_dir"/"${image}" ]
 then
  docker load -q -i  "$image_dir"/"${image}"
 fi
done

## rm container if exist.
if [ "$(docker ps -aq -f name=$container)" ]; then
    docker rm -f $container
fi

regArgs="-d --restart=always \
--net=host \
--name $container \
-v $certs_dir:/certs \
-v $VOLUME:/var/lib/registry \
-e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/sea.hub.crt \
-e REGISTRY_HTTP_TLS_KEY=/certs/sea.hub.key"

if [ -f $config ]; then
    regArgs="$regArgs \
    -v $config:/etc/docker/registry/config.yml"
fi

if [ -f $htpasswd ]; then
    docker run $regArgs \
            -v $htpasswd:/htpasswd \
            -e REGISTRY_AUTH=htpasswd \
            -e REGISTRY_AUTH_HTPASSWD_PATH=/htpasswd \
            -e REGISTRY_AUTH_HTPASSWD_REALM="Registry Realm" registry:2.7.1 || startRegistry
else
    docker run $regArgs registry:2.7.1 || startRegistry
fi