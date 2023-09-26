#!/usr/bin/env bash

# Copyright © 2022 Alibaba Group Holding Ltd.
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

# ==============================================================================

## Build script

# Build helper functions. These functions help to set up, save, and load the following variables:
#
#    GIT_TAG - the version number of sealer
#    MULTI_PLATFORM_BUILD - whether to build for all platforms (linux and darwin)

# Set GO111MODULE=on to enable Go Modules when using go mod to manage dependencies
export GO111MODULE=on
# Turn on command tracing so that each command is output when the script is run
set -x
# Get the absolute path of the current script and set the variable SEALER_ROOT to this value
SEALER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
# Set the variables THIS_PLATFORM_BIN and THIS_PLATFORM_ASSETS for use when building binaries and assets (such as tar.gz files)
export THIS_PLATFORM_BIN="${SEALER_ROOT}/_output/bin"
export THIS_PLATFORM_ASSETS="${SEALER_ROOT}/_output/assets"

# Fixed container dependency issues
# Link -> https://github.com/containers/image/pull/271/files
# For btrfs, we are currently only using overlays, so we don't need to include btrfs, otherwise we need to fix some lib issues
GO_BUILD_FLAGS="containers_image_openpgp exclude_graphdriver_devicemapper exclude_graphdriver_btrfs"

# Output debug information
debug() {
  timestamp=$(date +"[%m%d %H:%M:%S]")
  echo "[debug] ${timestamp} ${1-}" >&2
  shift
  for message; do
    echo "    ${message}" >&2
  done
}

# Get version information
get_version_vars() {
  GIT_VERSION=unknown
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

# Parameters used when building the program
# Used to pass parameters such as compilation time, version number, commit ID, etc.
ldflags() {
  local -a ldflags
  function add_ldflag() {
    local key=${1}
    local val=${2}
    # If you update these, also update the list component-base/version/def.bzl.
    ldflags+=(
      "-X '${SEALER_GO_PACKAGE}/pkg/version.${key}=${val}'"
    )
  }
  add_ldflag "buildDate" "$(date "+%FT %T %z")"
  if [[ -n ${GIT_COMMIT-} ]]; then
    add_ldflag "gitCommit" "${GIT_COMMIT}"
  fi

  if [[ -n ${GIT_VERSION-} ]]; then
    add_ldflag "gitVersion" "${GIT_VERSION}"
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}

# Package path for sealer source code
readonly SEALER_GO_PACKAGE=github.com/sealerio/sealer
# Platforms supported when building the program
readonly SEALER_SUPPORTED_PLATFORMS=(
  linux/amd64
  linux/arm64
)

# Check if the program needs to be built
check() {
  timestamp=$(date +"[%m%d %H:%M:%S]")
  ret=$1
  if [[ $ret -eq 0 ]]; then
    echo "[info] ${timestamp} ${2-} up to date."
  else
    echo "[err] ${timestamp} ${2-} is out of date. Please run $0"
    exit 1
  fi
}

# Function for building the program
build_binaries() {
  get_version_vars
  goldflags="${GOLDFLAGS=-s -w} $(ldflags)"
  osarch=${1-}_${2-}
  mkdir -p $THIS_PLATFORM_ASSETS
  tarFile="${GIT_VERSION}-${1-}-${2-}.tar.gz"

  debug "!!! build $osarch sealer"
  GOOS=${1-} GOARCH=${2-} go build -tags "${GO_BUILD_FLAGS}" -o $THIS_PLATFORM_BIN/sealer/$osarch/sealer -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/cmd/sealer/main.go
  check $? "build $osarch sealer"
  debug "output bin: $THIS_PLATFORM_BIN/sealer/$osarch/sealer"
  cd ${SEALER_ROOT}/_output/bin/sealer/$osarch/
  tar czf sealer-$tarFile sealer
  sha256sum sealer-$tarFile > sealer-$tarFile.sha256sum
  mv *.tar.gz*  $THIS_PLATFORM_ASSETS/
  debug "output tar.gz: $THIS_PLATFORM_ASSETS/sealer-$tarFile"
  debug "output sha256sum: $THIS_PLATFORM_ASSETS/sealer-$tarFile.sha256sum"

  debug "!!! build $osarch seautil"
  GOOS=${1-} GOARCH=${2-} go build -o $THIS_PLATFORM_BIN/seautil/$osarch/seautil -mod vendor -ldflags "$goldflags"  $SEALER_ROOT/cmd/seautil/main.go
  check $? "build $osarch seautil"
  debug "output bin: $THIS_PLATFORM_BIN/seautil/$osarch/seautil"
  cd ${SEALER_ROOT}/_output/bin/seautil/$osarch/
  tar czf seautil-$tarFile seautil
  sha256sum seautil-$tarFile >  seautil-$tarFile.sha256sum
  mv *.tar.gz*  $THIS_PLATFORM_ASSETS/
  debug "output tar.gz: $THIS_PLATFORM_ASSETS/seautil-$tarFile"
  debug "output sha256sum: $THIS_PLATFORM_ASSETS/seautil-$tarFile.sha256sum"
  debug ""
}

# Display help information
show_help() {
cat << EOF
Usage: $0 [-h] [-p PLATFORMS] [-a] [-b BINARIES]

Build Sealer binaries for one or more platforms.
    DOTO: I recommend using a Makefile for a more immersive experience

    -h, --help      display this help and exit

    -p, --platform  build binaries for the specified platform(s), e.g. linux/amd64 or linux/arm64. 
                    Multiple platforms should be separated by comma, e.g. linux/amd64,linux/arm64.
    
    -a, --all       build binaries for all supported platforms
    
    -b, --binary    build the specified binary/binaries, e.g. sealer or seautil.
                    Multiple binaries should be separated by comma, e.g. sealer,seautil.
                    (note: currently only supported in Makefile)

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      show_help
      exit 0
      ;;
    -p|--platform)
      shift
      PLATFORMS=$1
      ;;
    -a|--all)
      ALL_PLATFORMS=true
      ;;
    -b|--binary)
      shift
      BINARIES=$1
      ;;
    *)
      echo "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
  shift
done

debug "root dir: $SEALER_ROOT"
debug "build dir: $THIS_PLATFORM_BIN"

# Build binaries for the specified platforms
if [[ -n "$PLATFORMS" ]]; then
  IFS=',' read -ra PLATFORM_LIST <<< "$PLATFORMS"
  for platform in "${PLATFORM_LIST[@]}"; do
    OS=${platform%/*}
    ARCH=${platform##*/}
    build_binaries "$OS" "$ARCH"
  done
# Build binaries for all supported platforms
elif [[ "$ALL_PLATFORMS" = true ]]; then
  for platform in "${SEALER_SUPPORTED_PLATFORMS[@]}"; do
    OS=${platform%/*}
    ARCH=${platform##*/}
    build_binaries "$OS" "$ARCH"
  done
# Build the specified binaries
elif [[ -n "$BINARIES" ]]; then
  IFS=',' read -ra BINARY_LIST <<< "$BINARIES"
  for binary in "${BINARY_LIST[@]}"; do
    case "$binary" in
      sealer)
        build_binaries `go env GOOS` `go env GOARCH`
        ;;
      seautil)
        osarch=`go env GOOS`_`go env GOARCH`
        GOOS=`go env GOOS` GOARCH=`go env GOARCH` go build -o $THIS_PLATFORM_BIN/seautil/$osarch/seautil -mod vendor -ldflags "$(ldflags)" $SEALER_ROOT/cmd/seautil/main.go
        check $? "build seautil"
        debug "output bin: $THIS_PLATFORM_BIN/seautil/$osarch/seautil"
        ;;
      *)
        echo "Unknown binary: $binary"
        show_help
        exit 1
        ;;
    esac
  done
# Build all binaries for the current platform by default
else
  build_binaries `go env GOOS` `go env GOARCH`
fi