#!/bin/bash

rm -f /usr/bin/conntrack
rm -f /usr/bin/kubelet-pre-start.sh
rm -f /usr/bin/crictl
rm -f /usr/bin/kubeadm
rm -f /usr/bin/kubetcl
rm -f /usr/bin/kubelet
rm -f /usr/bin/containerd-rootless-setuptool.sh
rm -f /usr/bin/containerd-rootless.sh
rm -f /usr/bin/nerdctl
rm -f /usr/bin/seautil

rm -f /etc/sysctl.d/k8s.conf
rm -f /etc/systemd/system/kubelet.service
rm -rf /etc/systemd/system/kubelet.service.d
rm -rf /var/lib/kubelet/
rm -f /var/lib/kubelet/config.yaml
