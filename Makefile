Dirs=$(shell ls)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifneq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

lint: ## Run go lint against code.
	golangci-lint run -v ./...

style: fmt vet lint ## code style: fmt,vet,lint

build: clean ## build binaries by default
	@echo "build sealer and seautil bin"
	hack/build.sh

linux: clean ## build binaries for linux
	@echo "build sealer and seautil bin for linux"
	GOOS=linux GOARCH=amd64 hack/build.sh $(GitTag)

# sealer should be compiled in linux platform, otherwise there will be GraphDriver problem.
build-in-docker:
	docker run --rm -v ${PWD}:/usr/src/sealer -w /usr/src/sealer registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-build:v1 make linux

test-sealer:
	@echo "run e2e test for sealer bin"
	hack/test-sealer.sh

clean: ## clean
	@rm -rf _output

install-addlicense: ## check license if not exist install addlicense tools
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
filelicense: ## add license
	for file in ${Dirs} ; do \
		if [[  $$file != '_output' && $$file != 'docs' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' ]]; then \
			$(ADDLICENSE_BIN)  -y $(shell date +"%Y") -c "Alibaba Group Holding Ltd." -f hack/LICENSE_TEMPLATE ./$$file ; \
		fi \
    done


install-gosec: ## check license if not exist install addlicense tools
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
