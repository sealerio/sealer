#!/bin/bash

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

docker load -q -i ../images/registry.tar || true
docker rm $container -f || true
docker run -d --restart=always --net=host --name $container -v $VOLUME:/var/lib/registry registry:2.7.1 || startRegistry
