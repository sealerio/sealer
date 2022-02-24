# create nydusimages
./nydus-image create --blob-dir ./nydusdimages/blobs  --bootstrap ./rootfs.meta $1
#nydusdir need be scp
rm -rf $2
mkdir $2
nydusdconfig='
{\n
  "device": {\n
    "backend": {\n
      "type": "registry",\n
      "config": {\n
        "scheme": "http",\n
        "host": "'${3}':8000",\n
        "repo": "sealer"\n
      }\n
    },\n
    "cache": {\n
      "type": "blobcache",\n
      "config": {\n
        "work_dir": "./cache"\n
      }\n
    }\n
  },\n
  "mode": "direct",\n
  "digest_validate": false,\n
  "enable_xattr": true,\n
  "fs_prefetch": {\n
    "enable": true,\n
    "threads_count": 1,\n
    "merging_size": 131072,\n
    "bandwidth_rate":10485760\n
  }\n
}\n
'
echo -e ${nydusdconfig} > ./nydusd_scp_file/httpserver.json
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


