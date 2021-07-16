#!/bin/bash

systemctl stop docker
rm -rf /etc/docker/daemon.json
rm -rf /lib/systemd/system/docker.service
rm -rf /usr/lib/systemd/system/docker.service
systemctl daemon-reload

rm -f /usr/bin/conntrack
rm -f /usr/bin/kubelet-pre-start.sh
rm -f /usr/bin/containerd
rm -f /usr/bin/containerd-shim
rm -f /usr/bin/containerd-shim-runc-v2
rm -f /usr/bin/crictl
rm -f /usr/bin/ctr
rm -f /usr/bin/docker
rm -f /usr/bin/docker-init
rm -f /usr/bin/docker-proxy
rm -f /usr/bin/dockerd
rm -f /usr/bin/kubeadm
rm -f /usr/bin/kubetcl
rm -f /usr/bin/kubelet
rm -f /usr/bin/rootlesskit
rm -f /usr/bin/rootlesskit-docker-proxy
rm -f /usr/bin/runc
rm -f /usr/bin/vpnkit
rm -f /usr/bin/containerd-rootless-setuptool.sh
rm -f /usr/bin/containerd-rootless.sh
rm -f /usr/bin/nerdctl

rm -f /etc/sysctl.d/k8s.conf
rm -f /etc/systemd/system/kubelet.service
rm -rf /etc/systemd/system/kubelet.service.d
rm -rf /var/lib/kubelet/
rm -f /var/lib/kubelet/config.yaml
