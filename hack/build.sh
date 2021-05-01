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


export GO111MODULE=off

SEALER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
export THIS_PLATFORM_BIN="${SEALER_ROOT}/_output/bin"

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
  linux/arm
  linux/arm64
  darwin/amd64
  darwin/arm64
)

build_binaries() {
  goldflags="${GOLDFLAGS=-s -w} $(ldflags)"
  osarch=${1-}_${2-}
  go build -o $THIS_PLATFORM_BIN/sealer/$osarch/sealer -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/sealer/main.go
  debug "output bin: $THIS_PLATFORM_BIN/sealer/$osarch/sealer"

  go build -o $THIS_PLATFORM_BIN/seautil/$osarch/seautil -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/seautil/main.go
  debug "output bin: $THIS_PLATFORM_BIN/seautil/$osarch/seautil"

}

debug "root dir: $SEALER_ROOT"
debug "build dir: $THIS_PLATFORM_BIN"

#Multi platform
if [[ $MULTI_PLATFORM_BUILD ]]; then
   for platform in "${SEALER_SUPPORTED_PLATFORMS[@]}"; do
     OS=${platform%/*}
     ARCH=${platform##*/}
     build_binaries $OS $ARCH
   done;
else
  build_binaries `go env GOOS` `go env GOARCH`
fi

