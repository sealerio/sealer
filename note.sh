#!/bin/bash

# Author:wangfei
# Time:2020-12-06 11:17:39
# Name:./note.sh
# Description:This is a production script.
echo "### Usage" >> release_note.md
echo "
\`\`\`sh
# 下载并安装sealer, sealer是个golang的二进制工具，直接下载拷贝到bin目录即可, release页面也可下载
$ wget -c https://sealyun.oss-cn-beijing.aliyuncs.com/latest/sealer && \\
    chmod +x sealer && mv sealer /usr/bin
\`\`\`
" >> release_note.md
echo "### [amd64 下载地址]" >> release_note.md
echo "[oss 下载地址](http://sealer.oss-cn-beijing.aliyuncs.com/sealers/sealer-v${VERSION}-linux-amd64.tar.gz)" >> release_note.md
echo "[latest 版本 oss下载地址](https://sealer.oss-cn-beijing.aliyuncs.com/latest)" >> release_note.md
echo "### [arm64 下载地址]" >> release_note.md
echo "[oss 下载地址](http://sealer.oss-cn-beijing.aliyuncs.com/sealers/sealer-v${VERSION}-linux-arm64.tar.gz)" >> release_note.md
echo "[latest 版本 oss下载地址](https://sealer.oss-cn-beijing.aliyuncs.com/latest-arm64)" >> release_note.md