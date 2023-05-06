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
# Build management helpers.  These functions help to set, save and load the
#

GO := go
# ! go 1.8 some packages fail to be pulled out. You are advised to use the gvm switchover version of the tools toolkit
GO_SUPPORTED_VERSIONS ?= |1.17|1.18|1.19|1.20|

GO_LDFLAGS += -X $(VERSION_PACKAGE).gitVersion=${GIT_TAG} \
	-X $(VERSION_PACKAGE).gitCommit=${GIT_COMMIT} \
	-X $(VERSION_PACKAGE).GitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PACKAGE).buildDate=${BUILD_DATE} \
	-s -w		# -s -w deletes debugging information and symbol tables
# ifneq ($(DLV),)
# 	GO_BUILD_FLAGS += -gcflags "all=-N -l"
# 	LDFLAGS = ""
# endif
ifeq ($(DEBUG), 1)
	GO_BUILD_FLAGS += -gcflags "all=-N -l"
	GO_LDFLAGS=
endif
GO_BUILD_FLAGS += -tags "containers_image_openpgp netgo exclude_graphdriver_devicemapper static osusergo exclude_graphdriver_btrfs" -trimpath -ldflags "$(GO_LDFLAGS)"

# ifeq ($(GOOS),windows)
# 	GO_OUT_EXT := .exe
# endif

ifeq ($(ROOT_PACKAGE),)
	$(error the variable ROOT_PACKAGE must be set prior to including golang.mk, ->/Makefile)
endif

GOPATH ?= $(shell go env GOPATH)
ifeq ($(origin GOBIN), undefined)
	GOBIN := $(GOPATH)/bin
endif

# COMMANDS is Specify all files under ${ROOT_DIR}/cmd/ except those ending in.md
COMMANDS ?= $(filter-out %.md, $(wildcard ${ROOT_DIR}/cmd/*))
ifeq (${COMMANDS},)
  $(error Could not determine COMMANDS, set ROOT_DIR or run in source dir)
endif

# BINS is the name of each file in ${COMMANDS}, excluding the directory path
# If there are no files in ${COMMANDS}, or if all files end in.md, ${BINS} will be empty
BINS ?= $(foreach cmd,${COMMANDS},$(notdir ${cmd}))
ifeq (${BINS},)
  $(error Could not determine BINS, set ROOT_DIR or run in source dir)
endif

# TODO: EXCLUDE_TESTS variable, which contains the name of the package to be excluded from the test
EXCLUDE_TESTS := github.com/sealerio/sealer/test github.com/sealerio/sealer/pkg/logger

# ==============================================================================
# make: Nothing to be done build and build sub targets
# _output	$(OUTPUT_DIR)
# ├── assets
# │   ├── sealer-unknown-linux-amd64.tar.gz
# │   ├── sealer-unknown-linux-amd64.tar.gz.sha256sum
# │   ├── seautil-unknown-linux-amd64.tar.gz
# │   └── seautil-unknown-linux-amd64.tar.gz.sha256sum
# └── bin	$(BIN_DIR)
#     ├── sealer
#     │   └── linux_amd64
#     │   │   └── sealer
#     │   └── linux_arm64
#     │       └── sealer
#     └── seautil
#         └── linux_amd64
#             └── seautil
# COMMAND=sealer
# PLATFORM=linux_amd64
# OS=linux
# ARCH=amd64
# BINS=sealer seautil
# BIN_DIR=/root/workspaces/sealer/_output/bin
# ==============================================================================

## go.build.verify: Verify that a suitable version of Go exists
.PHONY: go.build.verify
go.build.verify:
	@echo "===========> Verify that a suitable version of Go exists, current version: $(shell $(GO) version)"
ifneq ($(shell $(GO) version | grep -q -E '\bgo($(GO_SUPPORTED_VERSIONS))\b' && echo 0 || echo 1), 0)
	$(error unsupported go version. Please make install one of the following supported version: '$(GO_SUPPORTED_VERSIONS)'.You can use make install. gvm to install gvm and switch to the specified version)
endif

## go.bin.%: Verify binary for specific platform
.PHONY: go.bin.%
go.bin.%:
	$(eval COMMAND := $(word 2,$(subst ., ,$*)))
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	@echo "===========> Verifying binary $(COMMAND) $(VERSION) for $(PLATFORM)"
	@if [ ! -f $(BIN_DIR)/$(PLATFORM)/$(COMMAND) ]; then echo $(MAKE) go.build PLATFORM=$(PLATFORM); fi

## go.build.%: Build binary for specific platform
.PHONY: go.build.%
go.build.%:
	$(eval COMMAND := $(word 2,$(subst ., ,$*)))
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	$(eval OS := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	@echo "COMMAND=$(COMMAND)"
	@echo "PLATFORM=$(PLATFORM)"
	@echo "OS=$(OS)"
	@echo "ARCH=$(ARCH)"
	@echo "BINS=$(BINS)"
	@echo "BIN_DIR=$(BIN_DIR)"
	@echo "===========> Building binary $(COMMAND) $(VERSION) for $(OS) $(ARCH)"
	@mkdir -p $(OUTPUT_DIR)/bin/$(OS)/$(ARCH)
	
#	@CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) $(GO) build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/bin/$(OS)/$(ARCH)/$(COMMAND)$(GO_OUT_EXT) $(ROOT_PACKAGE)/cmd/$(COMMAND)

## go.build: Build binaries
.PHONY: go.build
go.build: go.build.verify $(addprefix go.build., $(addprefix $(PLATFORM)., $(BINS)))
	@echo "COMMAND=$(COMMAND)"
	@echo "PLATFORM=$(PLATFORM)"
	@echo "OS=$(OS)"
	@echo "ARCH=$(ARCH)"
	@echo "===========> Building binary $(BINS) $(VERSION) for $(PLATFORM)"

## go.build.multiarch: Build multi-arch binaries
.PHONY: go.build.multiarch
go.build.multiarch: go.build.verify $(foreach p,$(PLATFORMS),$(addprefix go.build., $(addprefix $(p)., $(BINS))))

## go.linux.%: Build linux_amd64 OR linux_arm64 binaries
.PHONY: go.linux.%
go.linux.%:
	@echo "Building sealer and seautil binaries for Linux $*"
	@GOOS=linux GOARCH=$* $(BUILD_SCRIPTS) $(GIT_TAG)
	@echo "$(shell go version)"
	@echo "===========> Building binary for Linux $* $(BUILDAPP) *[Git Info]: $(VERSION)-$(GIT_TAG)-$(GIT_COMMIT)"

## go.lint: Run golangci to lint source codes
.PHONY: go.lint
go.lint: tools.verify.golangci-lint
	@echo "===========> Run golangci to lint source codes"
	@$(TOOLS_DIR)/golangci-lint run -c $(ROOT_DIR)/.golangci.yml $(ROOT_DIR)/...

## go.test: Run unit test
.PHONY: go.test
go.test:
	@$(GO) test ./...

## go.test.junit-report: Run unit test
.PHONY: go.test.junit-report
go.test.junit-report: tools.verify.go-junit-report
	@echo "===========> Run unit test > $(TMP_DIR)/report.xml"
	@$(GO) test -v -coverprofile=$(TMP_DIR)/coverage.out 2>&1 ./... | $(TOOLS_DIR)/go-junit-report -set-exit-code > $(TMP_DIR)/report.xml
	@sed -i '/mock_.*.go/d' $(TMP_DIR)/coverage.out
	@echo "===========> Test coverage of Go code is reported to $(TMP_DIR)/coverage.html by generating HTML"
	@$(GO) tool cover -html=$(TMP_DIR)/coverage.out -o $(TMP_DIR)/coverage.html

# .PHONY: go.test.junit-report
# go.test.junit-report: tools.verify.go-junit-report
# 	@echo "===========> Run unit test"
# 	@$(GO) test -race -cover -coverprofile=$(OUTPUT_DIR)/coverage.out \
# 		-timeout=10m -shuffle=on -short -v `go list ./pkg/hook/ |\
# 		egrep -v $(subst $(SPACE),'|',$(sort $(EXCLUDE_TESTS)))` 2>&1 | \
# 		tee >(go-junit-report --set-exit-code >$(OUTPUT_DIR)/report.xml)
# 	@sed -i '/mock_.*.go/d' $(OUTPUT_DIR)/coverage.out # remove mock_.*.go files from test coverage
# 	@$(GO) tool cover -html=$(OUTPUT_DIR)/coverage.out -o $(OUTPUT_DIR)/coverage.html

## go.test.cover: Run unit test with coverage
.PHONY: go.test.cover
go.test.cover: go.test.junit-report
	@touch $(TMP_DIR)/coverage.out
	@$(GO) tool cover -func=$(TMP_DIR)/coverage.out | \
		awk -v target=$(COVERAGE) -f $(ROOT_DIR)/scripts/coverage.awk

## go.format: Run unit test and format codes
.PHONY: go.format
go.format: tools.verify.golines tools.verify.goimports
	@echo "===========> Formating codes"
	@$(FIND) -type f -name '*.go' | $(XARGS) gofmt -s -w
	@$(FIND) -type f -name '*.go' | $(XARGS) $(TOOLS_DIR)/goimports -w -local $(ROOT_PACKAGE)
	@$(FIND) -type f -name '*.go' | $(XARGS) $(TOOLS_DIR)/golines -w --max-len=120 --reformat-tags --shorten-comments --ignore-generated .
	@$(GO) mod edit -fmt

## imports: task to automatically handle import packages in Go files using goimports tool
.PHONY: go.imports
go.imports: tools.verify.goimports
	@$(TOOLS_DIR)/goimports -l -w $(SRC)

## go.updates: Check for updates to go.mod dependencies
.PHONY: go.updates
go.updates: tools.verify.go-mod-outdated
	@$(GO) list -u -m -json all | go-mod-outdated -update -direct

## go.clean: Clean all builds directories and files
.PHONY: go.clean
go.clean:
	@echo "===========> Cleaning all builds OUTPUT_DIR($(OUTPUT_DIR)) AND BIN_DIR($(BIN_DIR))"
	@-rm -vrf $(OUTPUT_DIR) $(BIN_DIR)
	@echo "===========> End clean..."

## copyright.help: Show copyright help
.PHONY: go.help
go.help: scripts/make-rules/golang.mk
	$(call smallhelp)
