.DEFAULT_GOAL := help

Dirs=$(shell ls)
GIT_TAG := $(shell git describe --exact-match --tags --abbrev=0  2> /dev/null || echo untagged)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

TOOLS_DIR := hack/build.sh

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifneq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## fmt: Run go fmt against code.
fmt:
	go fmt ./...

## vet: Run go vet against code.
vet:
	go vet ./...

## lint: Run go lint against code.
lint:
	golangci-lint run -v ./...

## style: code style: fmt,vet,lint
style: fmt vet lint

## build: Build binaries by default
build: clean
	@echo "build sealer and seautil bin"
	$(TOOLS_DIR)

## linux: Build binaries for Linux
linux: clean
	@echo "Building sealer and seautil binaries for Linux (amd64)"
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(TOOLS_DIR) $(GIT_TAG)

## linux-amd64: Build binaries for Linux (amd64)
linux-amd64: clean
	@echo "Building sealer and seautil binaries for Linux (amd64)"
	GOOS=linux GOARCH=amd64 $(TOOLS_DIR) $(GIT_TAG)

## linux-arm64: Build binaries for Linux (arm64)
linux-arm64: clean
	@echo "Building sealer and seautil binaries for Linux (arm64)"
	GOOS=linux GOARCH=arm64 $(TOOLS_DIR) $(GIT_TAG)

## build-in-docker: sealer should be compiled in linux platform, otherwise there will be GraphDriver problem.
build-in-docker:
	docker run --rm -v ${PWD}:/usr/src/sealer -w /usr/src/sealer registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-build:v1 make linux

## clean: Remove all files that are created by building. 
.PHONY: clean
clean:
	@echo "===========> Cleaning all build output"
	@-rm -rf _output

## install-addlicense: check license if not exist install addlicense tools
install-addlicense:
ifeq (, $(shell which addlicense))
	@{ \
	set -e ;\
	LICENSE_TMP_DIR=$$(mktemp -d) ;\
	cd $$LICENSE_TMP_DIR ;\
	go mod init tmp ;\
	go get -v github.com/google/addlicense ;\
	rm -rf $$LICENSE_TMP_DIR ;\
	}
ADDLICENSE_BIN=$(GOBIN)/addlicense
else
ADDLICENSE_BIN=$(shell which addlicense)
endif

filelicense: SHELL:=/bin/bash
## filelicense: add license
filelicense:
	for file in ${Dirs} ; do \
		if [[  $$file != '_output' && $$file != 'docs' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' ]]; then \
			$(ADDLICENSE_BIN)  -y $(shell date +"%Y") -c "Alibaba Group Holding Ltd." -f hack/LICENSE_TEMPLATE ./$$file ; \
		fi \
    done


## install-gosec: check license if not exist install addlicense tools
install-gosec:
ifeq (, $(shell which gosec))
	@{ \
	set -e ;\
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOBIN) v2.2.0 ;\
	}
GOSEC_BIN=$(GOBIN)/gosec
else
GOSEC_BIN=$(shell which gosec)
endif

gosec: install-gosec
	$(GOSEC_BIN) ./...


install-deepcopy-gen:
ifeq (, $(shell which deepcopy-gen))
	@{ \
	set -e ;\
	LICENSE_TMP_DIR=$$(mktemp -d) ;\
	cd $$LICENSE_TMP_DIR ;\
	go mod init tmp ;\
	go get -v k8s.io/code-generator/cmd/deepcopy-gen ;\
	rm -rf $$LICENSE_TMP_DIR ;\
	}
DEEPCOPY_BIN=$(GOBIN)/deepcopy-gen
else
DEEPCOPY_BIN=$(shell which deepcopy-gen)
endif

HEAD_FILE := hack/boilerplate.go.txt
INPUT_DIR := github.com/sealerio/sealer/types/api
deepcopy:install-deepcopy-gen
	$(DEEPCOPY_BIN) \
      --input-dirs="$(INPUT_DIR)/v1" \
      -O zz_generated.deepcopy   \
      --go-header-file "$(HEAD_FILE)" \
      --output-base "${GOPATH}/src"
	$(DEEPCOPY_BIN) \
	  --input-dirs="$(INPUT_DIR)/v2" \
	  -O zz_generated.deepcopy   \
	  --go-header-file "$(HEAD_FILE)" \
	  --output-base "${GOPATH}/src"

## help: Display help information
help: Makefile
	@echo ""
	@echo "Usage:" "\n"
	@echo "  make [target]" "\n"
	@echo "Targets:" "\n" ""
	@awk -F ':|##' '/^[^\.%\t][^\t]*:.*##/{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST) | sort
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
