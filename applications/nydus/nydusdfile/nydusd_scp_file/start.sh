rm -rf $1
mkdir -p $1
rm -rf nydusdfs
mkdir -p nydusdfs
cp nydusd /usr/bin/nydusd
nydusdcmd="#!/bin/bash\n/usr/bin/nydusd --thread-num 10 --log-level debug --mountpoint $(pwd)/nydusdfs --apisock $(pwd)/nydusd.sock --id sealer --bootstrap $(pwd)/rootfs.meta --config $(pwd)/httpserver.json --supervisor $(pwd)/supervisor.sock"
echo -e ${nydusdcmd} > /var/lib/sealer/nydusd.sh
chmod +x /var/lib/sealer/nydusd.sh
cp nydusd.service /etc/systemd/system/
systemctl enable nydusd.service
systemctl restart nydusd.service
rm -rf upper
rm -rf work
mkdir -p upper
mkdir -p work
sleep 0.5
mount -t overlay overlay -o lowerdir=$(pwd)/nydusdfs,upperdir=$(pwd)/upper,workdir=$(pwd)/work $1 -o index=off
