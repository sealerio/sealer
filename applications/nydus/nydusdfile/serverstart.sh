# create nydusimages
./nydus-image create --blob-dir ./nydusdimages/blobs  --bootstrap ./rootfs.meta $1
#nydusdir need be scp
rm -rf $2
mkdir $2
cp ./nydusd_scp_file/* $2
cp ./rootfs.meta $2
# nydusd_http_server.service
service="[Unit]\n
Description=sealer nydusd rootfs\n
[Service]\n
TimeoutStartSec=3\n
Environment=\"ROCKET_CONFIG=$(pwd)/Rocket.toml\"\n
ExecStart=$(pwd)/nydusdserver $(pwd)/nydusdimages\n
Restart=always\n
[Install]\n
WantedBy=multi-user.target"
echo -e ${service} > /etc/systemd/system/nydusd_http_server.service
# start nydusd_http_server.service
systemctl enable nydusd_http_server.service
systemctl restart nydusd_http_server.service


