#!/bin/bash

# Copyright © 2021 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Author:wangfei
# Time:2021-11-15
# Name:./note.sh
# Description:This is a production script.
echo "
### Usage" >> release_note.md
echo "
\`\`\`sh
# Download and install sealer. Sealer is a binary tool of golang. You can download and unzip it directly to the bin directory, and the release page can also be downloaded
$ wget -c https://sealer.oss-cn-beijing.aliyuncs.com/sealers/sealer-v${VERSION}-linux-amd64.tar.gz && \\
      tar -xvf sealer-v${VERSION}-linux-amd64.tar.gz -C /usr/bin
\`\`\`
" >> release_note.md
echo "### [amd64 Download address]" >> release_note.md
echo "[OSS Download address](http://sealer.oss-cn-beijing.aliyuncs.com/sealers/sealer-v${VERSION}-linux-amd64.tar.gz)" >> release_note.md
echo "### [arm64 Download address]" >> release_note.md
echo "[OSS Download address](http://sealer.oss-cn-beijing.aliyuncs.com/sealers/sealer-v${VERSION}-linux-arm64.tar.gz)" >> release_note.md