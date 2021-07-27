#!/bin/bash

systemctl stop docker
rm -rf /etc/docker/daemon.json
rm -rf /lib/systemd/system/docker.service
rm -rf /usr/lib/systemd/system/docker.service
systemctl daemon-reload

rm -f /usr/bin/containerd
rm -f /usr/bin/containerd-shim
rm -f /usr/bin/ctr
rm -f /usr/bin/docker
rm -f /usr/bin/docker-init
rm -f /usr/bin/docker-proxy
rm -f /usr/bin/dockerd
rm -f /usr/bin/runc
