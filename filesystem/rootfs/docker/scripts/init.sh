#!/bin/bash


STORAGE=${1:-/var/lib/docker}
REGISTRY_DOMAIN=${2-sea.hub}
REGISTRY_PORT=${3-5000}


# Install docker
chmod a+x docker.sh
#./docker.sh  /var/docker/lib  127.0.0.1
bash docker.sh ${STORAGE} ${REGISTRY_DOMAIN} $REGISTRY_PORT

chmod a+x init-kube.sh

bash init-kube.sh
