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
# Makefile helper functions for common tasks
#

SHELL := /bin/bash
GO:=go
DIRS=$(shell ls)
DEBUG ?= 0
GIT_TAG := $(shell git describe --exact-match --tags --abbrev=0  2> /dev/null || echo untagged)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD || echo "0.0.0")
BUILD_DATE=$(shell date '+%FT %T %z')	# "buildDate":"2023-03-31T 20:05:43 +0800"

# include the common makefile
COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

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
BIN_DIR := $(OUTPUT_DIR)/bin
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

# Check if the tree is dirty. default to dirty(maybe u should commit?)
GIT_TREE_STATE:="dirty"
ifeq (, $(shell git status --porcelain 2>/dev/null))
	GIT_TREE_STATE="clean"
endif
GIT_COMMIT:=$(shell git rev-parse HEAD)

# Minimum test coverage
ifeq ($(origin COVERAGE),undefined)
COVERAGE := 60
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# platforms: The OS must be linux when building docker images
PLATFORMS ?= linux_amd64 linux_arm64
# TODO(sealer?): The OS can be linux/windows/darwin when building binaries? if only support linux
# PLATFORMS ?= darwin_amd64 windows_amd64 linux_amd64 linux_arm64

# only support linux
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
# TODO: Whether you need to join utils?
FIND := find . ! -path './utils/*' ! -path './vendor/*'
XARGS := xargs -r

# Linux command settings-CODE DIRS Copyright
# $$file != '_output' && $$file != 'docs' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' 
CODE_DIRS := $(ROOT_DIR)/pkg $(ROOT_DIR)/cmd $(ROOT_DIR)/test $(ROOT_DIR)/build $(ROOT_DIR)/scripts $(ROOT_DIR)/utils $(ROOT_DIR)/common
FINDS := find $(CODE_DIRS)

# Makefile settings: Select different behaviors by determining whether V option is set
ifeq ($(origin V), undefined) # ifndef V
MAKEFLAGS += --no-print-directory
endif

# COMMA: Concatenate multiple strings to form a list of strings
COMMA := ,
# SPACE: Used to separate strings
SPACE :=
# SPACE: Replace multiple consecutive Spaces with a single space
SPACE +=

# ==============================================================================
# Makefile helper functions for common tasks

# Help information for the makefile package
define makehelp
	@printf "\n\033[1mUsage: make <TARGETS> <OPTIONS> ...\033[0m\n\n\\033[1mTargets:\\033[0m\n\n"
	@sed -n 's/^##//p' $< | awk -F':' '{printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}' | sed -e 's/^/ /'
	@printf "\n\033[1m$$USAGE_OPTIONS\033[0m\n"
endef

# Here are some examples of builds
define MAKEFILE_EXAMPLE
# make build BINS=sealer                                         Only a single sealer binary is built 
# make -j $(nproc) all                                           Run tidy gen add-copyright format lint cover build concurrently
# make gen                                                       Generate all necessary files
# make deepcopy                                                  Generate deepcopy code
# make verify-copyright                                          Verify the license headers for all files
# make install-deepcopy-gen                                      Install deepcopy-gen tools if the license is missing
# make build BINS=sealer V=1 DEBUG=1                             Build debug binaries for only sealer
# make build.multiarch PLATFORMS="linux_arm64 linux_amd64" V=1   Build binaries for both platforms
endef
export MAKEFILE_EXAMPLE

# Define all help functions	@printf "\n\033[1mCurrent sealer version information: $(shell sealer version):\033[0m\n\n"
define makeallhelp
	@printf "\n\033[1mMake example:\033[0m\n\n"
	$(call MAKEFILE_EXAMPLE)
	@printf "\n\033[1mAriables:\033[0m\n\n"
	@echo "  DEBUG: $(DEBUG)"
	@echo "  BINS: $(BINS)"
	@echo "  PLATFORMS: $(PLATFORMS)"
	@echo "  V: $(V)"
endef

# Help information for other makefile packages
CUT_OFF?="---------------------------------------------------------------------------------"
HELP_NAME:=$(shell basename $(MAKEFILE_LIST))
define smallhelp
	@sed -n 's/^##//p' $< | awk -F':' '{printf "\033[36m%-35s\033[0m %s\n", $$1, $$2}' | sed -e 's/^/ /'
	@echo $(CUT_OFF)
endef