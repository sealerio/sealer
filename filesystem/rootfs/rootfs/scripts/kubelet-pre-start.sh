#!/bin/bash
# Open ipvs
modprobe -- ip_vs
modprobe -- ip_vs_rr
modprobe -- ip_vs_wrr
modprobe -- ip_vs_sh
# 1.20 need ope br_netfilter
modprobe -- br_netfilter
modprobe -- bridge
version_ge(){
    test "$(echo "$@" | tr ' ' '\n' | sort -rV | head -n 1)" == "$1"
}
disable_selinux(){
    if [ -s /etc/selinux/config ] && grep 'SELINUX=enforcing' /etc/selinux/config; then
        sed -i 's/SELINUX=enforcing/SELINUX=disabled/g' /etc/selinux/config
        setenforce 0
    fi
}

kernel_version=$(uname -r | cut -d- -f1)
if version_ge "${kernel_version}" 4.19; then
  modprobe -- nf_conntrack
else
  modprobe -- nf_conntrack_ipv4
fi

cat <<EOF >  /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.conf.all.rp_filter=0
EOF
sysctl --system
sysctl -w net.ipv4.ip_forward=1
# systemctl stop firewalld && systemctl disable firewalld
swapoff -a
disable_selinux
exit 0
