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

mkdir -p $VOLUME || true

startRegistry() {
    n=1
    while (( $n <= 3 ))
    do
        echo "attempt to start registry"
        (docker start $container && break) || true
        (( n++ ))
        sleep 3
    done
}

[ -f ../images/registry.tar  ] && docker load -q -i ../images/registry.tar 

## rm container if exist.
if [ "$(docker ps -aq -f name=$container)" ]; then
    docker rm -f $container
fi

config=$(dirname "$(pwd)")'/etc/registry_config.yaml'
certs_dir=$(cd ../;pwd)'/certs'

if [ -f $config ]; then
    docker run -d --restart=always --net=host --name $container -v $certs_dir:/certs -v $VOLUME:/var/lib/registry registry:2.7.1 -v $config:/etc/docker/registry/config.yml -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/sea.hub.crt -e REGISTRY_HTTP_TLS_KEY=/certs/sea.hub.key || startRegistry
else
    docker run -d --restart=always --net=host --name $container -v $certs_dir:/certs -v $VOLUME:/var/lib/registry registry:2.7.1 -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/sea.hub.crt -e REGISTRY_HTTP_TLS_KEY=/certs/sea.hub.key || startRegistry
fi