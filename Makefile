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
	@echo "build sealer and sealutil bin"
	hack/build.sh

linux: clean ## build binaries for linux
	@echo "build sealer and sealutil bin for linux"
	GOOS=linux GOARCH=amd64 hack/build.sh $(GitTag)

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

license: SHELL:=/bin/bash
license: ## add license
	for file in ${Dirs} ; do \
		if [[  $$file != '_output' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' ]]; then \
			$(ADDLICENSE_BIN)  -y $(shell date +"%Y") -c "Alibaba Group Holding Ltd." -f LICENSE_TEMPLATE ./$$file ; \
		fi \
    done
