#!/bin/bash

# Copyright 2021 cuisongliu@qq.com.
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

# -----------------------------------------------------------------------------
# Build management helpers.  These functions help to set, save and load the
# following variables:
#
#    GIT_TAG - The version for sealer.
#    MULTI_PLATFORM_BUILD -  Need build all platform.(linux and darwin)


export GO111MODULE=on

SEALER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
export THIS_PLATFORM_BIN="${SEALER_ROOT}/_output/bin"
export THIS_PLATFORM_ASSETS="${SEALER_ROOT}/_output/assets"

debug() {
  timestamp=$(date +"[%m%d %H:%M:%S]")
  echo "[debug] ${timestamp} ${1-}" >&2
  shift
  for message; do
    echo "    ${message}" >&2
  done
}

get_version_vars() {
  GIT_VERSION=latest
  if [[ $GIT_TAG ]]; then
    GIT_VERSION=$GIT_TAG
  fi
  GIT_COMMIT=`git rev-parse --short HEAD || true`
  if [[ -z $GIT_COMMIT ]]; then
    GIT_COMMIT="0.0.0"
  fi

  debug "version: $GIT_VERSION"
  debug "commit id: $GIT_COMMIT"
}

ldflags() {
  get_version_vars

  local -a ldflags
  function add_ldflag() {
    local key=${1}
    local val=${2}
    # If you update these, also update the list component-base/version/def.bzl.
    ldflags+=(
      "-X '${SEALER_GO_PACKAGE}/version.${key}=${val}'"
    )
  }
  add_ldflag "buildDate" "$(date "+%F %T")"
  if [[ -n ${GIT_COMMIT-} ]]; then
    add_ldflag "gitCommit" "${GIT_COMMIT}"
  fi

  if [[ -n ${GIT_VERSION-} ]]; then
    add_ldflag "gitVersion" "${GIT_VERSION}"
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}

readonly SEALER_GO_PACKAGE=github.com/alibaba/sealer
# The server platform we are building on.
readonly SEALER_SUPPORTED_PLATFORMS=(
  linux/amd64
  linux/arm64
)

build_binaries() {
  goldflags="${GOLDFLAGS=-s -w} $(ldflags)"
  osarch=${1-}_${2-}
  mkdir -p $THIS_PLATFORM_ASSETS
  tarFile="${3-}-${1-}-${2-}.tar.gz"

  debug "!!! build $osarch sealer"
  GOOS=${1-} GOARCH=${2-} go build -o $THIS_PLATFORM_BIN/sealer/$osarch/sealer -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/sealer/main.go
  debug "output bin: $THIS_PLATFORM_BIN/sealer/$osarch/sealer"
  cd ${SEALER_ROOT}/_output/bin/sealer/$osarch/
  tar czf sealer-$tarFile sealer
  sha256sum sealer-$tarFile > sealer-$tarFile.sha256sum
  mv *.tar.gz*  $THIS_PLATFORM_ASSETS/
  debug "output tar.gz: $THIS_PLATFORM_ASSETS/sealer-$tarFile"
  debug "output sha256sum: $THIS_PLATFORM_ASSETS/sealer-$tarFile.sha256sum"

  debug "!!! build $osarch seautil"
  GOOS=${1-} GOARCH=${2-} go build -o $THIS_PLATFORM_BIN/seautil/$osarch/seautil -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/seautil/main.go
  debug "output bin: $THIS_PLATFORM_BIN/seautil/$osarch/seautil"
  cd ${SEALER_ROOT}/_output/bin/seautil/$osarch/
  tar czf seautil-$tarFile seautil
  sha256sum seautil-$tarFile >  seautil-$tarFile.sha256sum
  mv *.tar.gz*  $THIS_PLATFORM_ASSETS/
  debug "output tar.gz: $THIS_PLATFORM_ASSETS/seautil-$tarFile"
  debug "output sha256sum: $THIS_PLATFORM_ASSETS/seautil-$tarFile.sha256sum"

}

debug "root dir: $SEALER_ROOT"
debug "build dir: $THIS_PLATFORM_BIN"

#Multi platform
if [[ $MULTI_PLATFORM_BUILD ]]; then
   for platform in "${SEALER_SUPPORTED_PLATFORMS[@]}"; do
     OS=${platform%/*}
     ARCH=${platform##*/}
     build_binaries $OS $ARCH ${1-0.0.0}
   done;
else
  build_binaries `go env GOOS` `go env GOARCH` ${1-0.0.0}
fi

