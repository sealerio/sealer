#!/bin/bash
command_exists() {
    command -v "$@" > /dev/null 2>&1
}
get_distribution() {
    lsb_dist=""
    # Every system that we officially support has /etc/os-release
    if [ -r /etc/os-release ]; then
    	lsb_dist="$(. /etc/os-release && echo "$ID")"
    fi
    # Returning an empty string here should be alright since the
    # case statements don't act unless you provide an actual value
    echo "$lsb_dist"
}
set -x
storage=${1:-/var/lib/docker}
mkdir -p $storage
if ! command_exists docker; then
  lsb_dist=$( get_distribution )
  lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"
  echo "current system is $lsb_dist"
  case "$lsb_dist" in
    ubuntu|deepin|debian|raspbian|kylin)
    	cp ../etc/docker.service /lib/systemd/system/docker.service
    ;;
  	centos|rhel|ol|sles|kylin|neokylin)
			cp ../etc/docker.service /usr/lib/systemd/system/docker.service
		;;
    alios)
      ip link add name docker0 type bridge
      ip addr add dev docker0 172.17.0.1/16
    	cp ../etc/docker.service /usr/lib/systemd/system/docker.service
    ;;
    *)
			echo "unknown system to use /lib/systemd/system/docker.service"
			cp ../etc/docker.service /lib/systemd/system/docker.service
    ;;
  esac

  [ -d  /etc/docker/ ] || mkdir /etc/docker/  -p

  chmod -R 755 ../cri
  tar -zxvf ../cri/docker.tar.gz -C /usr/bin
  chmod a+x /usr/bin
  chmod a+x /usr/bin/docker
  chmod a+x /usr/bin/dockerd
  systemctl enable docker.service
  systemctl restart docker.service
  cp ../etc/daemon.json /etc/docker
  sed -i "s/$2:5000/$2:$3/g" /etc/docker/daemon.json
fi
systemctl daemon-reload
systemctl restart docker.service

cgroupDriver=$(docker info|grep Cg)
driver=${cgroupDriver##*: }
echo "driver is ${driver}"
export criDriver=${driver}
