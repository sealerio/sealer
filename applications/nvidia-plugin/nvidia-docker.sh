#!/bin/bash
tar -xvf ./nvidia-docker.tar
cp ./nvidia-docker/nvidia-container-* /usr/bin/
cp ./nvidia-docker/libnvidia-container.so.1 /usr/lib64/
cp ./nvidia-docker/libseccomp.so.2 /lib64/
chmod a+x /usr/bin/nvidia-container-*
cp ./nvidia-docker/daemon.json /etc/docker
systemctl restart docker