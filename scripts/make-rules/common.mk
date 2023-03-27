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

SHELL := /bin/bash
DIRS=$(shell ls)

# include the common makefile
COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

# ROOT_DIR: root directory of the code base
ifeq ($(origin ROOT_DIR),undefined)
ROOT_DIR := $(abspath $(shell cd $(COMMON_SELF_DIR)/../.. && pwd -P))
endif

# OUTPUT_DIR: The directory where the build output is stored.
ifeq ($(origin OUTPUT_DIR),undefined)
OUTPUT_DIR := $(ROOT_DIR)/_output
$(shell mkdir -p $(OUTPUT_DIR))
endif

# BIN_DIR: Directory where executable files are stored.
ifeq ($(origin BIN_DIR),undefined)
BIN_DIR := $(ROOT_DIR)/bin
$(shell mkdir -p $(BIN_DIR))
endif

# TOOLS_DIR: The directory where tools are stored for build and testing.
ifeq ($(origin TOOLS_DIR),undefined)
TOOLS_DIR := $(ROOT_DIR)/tools
$(shell mkdir -p $(TOOLS_DIR))
endif

# TMP_DIR: directory where temporary files are stored.
ifeq ($(origin TMP_DIR),undefined)
TMP_DIR := $(ROOT_DIR)/tmp
$(shell mkdir -p $(TMP_DIR))
endif

ifeq ($(origin VERSION), undefined)
VERSION := $(shell git describe --abbrev=0 --dirty --always --tags | sed 's/-/./g')
endif

# platforms: The OS must be linux when building docker images
PLATFORMS ?= linux_amd64 linux_arm64
# TODO(sealer?): The OS can be linux/windows/darwin when building binaries? if only support linux
# PLATFORMS ?= darwin_amd64 windows_amd64 linux_amd64 linux_arm64
GOOS=linux

# set a specific PLATFORM, defaults to the host platform
ifeq ($(origin PLATFORM), undefined)
	ifeq ($(origin GOARCH), undefined)
		GOARCH := $(shell go env GOARCH)
	endif
	ifeq ($(origin GOARCH), undefined)
		GOARCH := $(shell go env GOARCH)
	endif
	PLATFORM := $(GOOS)_$(GOARCH)
	# Use linux as the default OS when building images
	IMAGE_PLAT := linux_$(GOARCH)
else
	# such as: PLATFORM = linux_amd64
	GOOS := $(word 1, $(subst _, ,$(PLATFORM)))
	GOARCH := $(word 2, $(subst _, ,$(PLATFORM)))
	IMAGE_PLAT := $(PLATFORM)
endif


# Linux command settings
CODE_DIRS := $(ROOT_DIR)/pkg $(ROOT_DIR)/cmd $(ROOT_DIR)/test $(ROOT_DIR)/staging
FIND := find $(CODE_DIRS)
