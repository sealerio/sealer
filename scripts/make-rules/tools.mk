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
# Makefile helper functions for tools(https://github.com/avelino/awesome-go) -> DIR: {TOOT_DIR}/tools | (go >= 1.19)
#

BUILD_TOOLS ?= golangci-lint goimports addlicense deepcopy-gen conversion-gen ginkgo

.PHONY: tools.install
## tools.install: Install all tools
tools.install: $(addprefix tools.install., $(BUILD_TOOLS))
 
.PHONY: tools.install.%
## tools.install.%: Install a single tool
tools.install.%:
	@echo "===========> Installing $,The default installation path is $(GOBIN)/$*"
	@$(MAKE) install.$*
	@echo "===========> $* installed successfully"

.PHONY: tools.verify.%
## tools.verify.%: Check if a tool is installed and install it
tools.verify.%:
	@echo "===========> Verifying $* is installed"
	@if [ ! -f $(TOOLS_DIR)/$* ]; then GOBIN=$(TOOLS_DIR) $(MAKE) tools.install.$*; fi

.PHONY:  
## install.golangci-lint: Install golangci-lint
install.golangci-lint:
	@echo "===========> Installing golangci-lint,The default installation path is $(GOBIN)/golangci-lint"
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
#	@golangci-lint completion bash > $(HOME)/.golangci-lint.bash
#	@if ! grep -q .golangci-lint.bash $(HOME)/.bashrc; then echo "source \$$HOME/.golangci-lint.bash" >> $(HOME)/.bashrc; fi

.PHONY: install.goimports
## install.goimports: Install goimports, used to format go source files
install.goimports:
	@echo "===========> Installing goimports,The default installation path is $(GOBIN)/goimports"
	@$(GO) install golang.org/x/tools/cmd/goimports@latest

# Actions path: https://github.com/sealerio/sealer/tree/main/.github/workflows/go.yml#L37-L50
.PHONY: install.addlicense
## install.addlicense: Install addlicense, used to add license header to source files
install.addlicense:
	@$(GO) install github.com/google/addlicense@latest

.PHONY: install.deepcopy-gen
## install.deepcopy-gen: Install deepcopy-gen, used to generate deep copy functions
install.deepcopy-gen:
	@$(GO) install k8s.io/code-generator/cmd/deepcopy-gen@latest

.PHONY: install.conversion-gen
## install.conversion-gen: Install conversion-gen, used to generate conversion functions
install.conversion-gen:
	@$(GO) install k8s.io/code-generator/cmd/conversion-gen@latest

.PHONY: install.ginkgo
## install.ginkgo: Install ginkgo to run a single test or set of tests
install.ginkgo:
	@echo "===========> Installing ginkgo,The default installation path is $(GOBIN)/ginkgo"
	@$(GO) install github.com/onsi/ginkgo/ginkgo@v1.16.2

# ==============================================================================
# Tools that might be used include go gvm
#

.PHONY: install.go-junit-report
## go-junit-report: Install go-junit-report, used to convert go test output to junit xml
install.go-junit-report:
	@$(GO) install github.com/jstemmer/go-junit-report@latest

.PHONY: install.kube-score
## install.kube-score: Install kube-score, used to check kubernetes yaml files
install.kube-score:
	@$(GO) install github.com/zegl/kube-score/cmd/kube-score@latest

.PHONY: install.go-gitlint
## Install go-gitlint: Install go-gitlint, used to check git commit message
install.go-gitlint:
	@$(GO) install github.com/marmotedu/go-gitlint/cmd/go-gitlint@latest

.PHONY: install.gsemver
## install.gsemver: Install gsemver, used to generate semver
install.gsemver:
	@$(GO) install github.com/arnaud-deprez/gsemver@latest

.PHONY: install.git-chglog
## install.git-chglog: Install git-chglog, used to generate changelog
install.git-chglog:
	@$(GO) install github.com/git-chglog/git-chglog/cmd/git-chglog@latest

.PHONY: install.github-release
## install.github-release: Install github-release, used to create github release
install.github-release:
	@$(GO) install github.com/github-release/github-release@latest

.PHONY: install.gvm
## install.gvm: Install gvm, gvm is a Go version manager, built on top of the official go tool.
install.gvm:
	@echo "===========> Installing gvm,The default installation path is ~/.gvm/scripts/gvm"
	@bash < <(curl -s -S -L https://raw.gitee.com/moovweb/gvm/master/binscripts/gvm-installer)
	@$(shell source /root/.gvm/scripts/gvm)

.PHONY: install.coscli
## install.coscli: Install coscli. COSCLI is a command line tool for Tencent Cloud Object Storage (COS)
install.coscli:
	@wget -q https://github.com/tencentyun/coscli/releases/download/v0.10.2-beta/coscli-linux -O ${HOME}/bin/coscli
	@chmod +x ${HOME}/bin/coscli

.PHONY: install.coscmd
## install.coscmd: Install coscmd, used to upload files to Tencent Cloud Object Storage (COS)
install.coscmd:
	@if which pip &>/dev/null; then pip install coscmd; else pip3 install coscmd; fi

.PHONY: install.golines
## install.golines: Install golines, used to format long lines
install.golines:
	@$(GO) install github.com/segmentio/golines@latest

.PHONY: install.go-mod-outdated
## install.go-mod-outdated: Install go-mod-outdated, used to check outdated dependencies
install.go-mod-outdated:
	@$(GO) install github.com/psampaz/go-mod-outdated@latest

.PHONY: install.mockgen
## install.mockgen: Install mockgen, used to generate mock functions
install.mockgen:
	@$(GO) install github.com/golang/mock/mockgen@latest

.PHONY: install.gotests
## install.gotests: Install gotests, used to generate test functions
install.gotests:
	@$(GO) install github.com/cweill/gotests/gotests@latest

.PHONY: install.protoc-gen-go
## install.protoc-gen-go: Install protoc-gen-go, used to generate go source files from protobuf files
install.protoc-gen-go:
	@$(GO) install github.com/golang/protobuf/protoc-gen-go@latest

.PHONY: install.cfssl
## install.cfssl: Install cfssl, used to generate certificates
install.cfssl:
	@$(ROOT_DIR)/scripts/install/install.sh iam::install::install_cfssl

.PHONY: install.depth
## install.depth: Install depth, used to check dependency tree
install.depth:
	@$(GO) install github.com/KyleBanks/depth/cmd/depth@latest

.PHONY: install.go-callvis
## install.go-callvis: Install go-callvis, used to visualize call graph
install.go-callvis:
	@$(GO) install github.com/ofabry/go-callvis@latest

.PHONY: install.gothanks
## install.gothanks: Install gothanks, used to thank go dependencies
install.gothanks:
	@$(GO) install github.com/psampaz/gothanks@latest

.PHONY: install.richgo
## install.richgo: Install richgo
install.richgo:
	@$(GO) install github.com/kyoh86/richgo@latest

.PHONY: install.rts
## install.rts: Install rts
install.rts:
	@$(GO) install github.com/galeone/rts/cmd/rts@latest

.PHONY: install.codegen
## install.codegen: Install code generator, used to generate code
install.codegen:
	@$(GO) install ${ROOT_DIR}/tools/codegen/codegen.go

.PHONY: tools.help
## tools.help: Display help information about the tools package
tools.help: scripts/make-rules/tools.mk
	$(call smallhelp)