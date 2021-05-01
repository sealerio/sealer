.PHONY: fmt vet lint default
GO_RELEASE_TAGS := $(shell go list -f ':{{join (context.ReleaseTags) ":"}}:' runtime)

# Only use the `-race` flag on newer versions of Go (version 1.3 and newer)
ifeq (,$(findstring :go1.3:,$(GO_RELEASE_TAGS)))
	RACE_FLAG :=
else
	RACE_FLAG := -race -cpu 1,2,4
endif

# Run `go vet` on Go 1.12 and newer. For Go 1.5-1.11, use `go tool vet`
ifneq (,$(findstring :go1.12:,$(GO_RELEASE_TAGS)))
	GO_VET := go vet \
		-atomic \
		-bool \
		-copylocks \
		-nilfunc \
		-printf \
		-rangeloops \
		-unreachable \
		-unsafeptr \
		-unusedresult \
		.
else ifneq (,$(findstring :go1.5:,$(GO_RELEASE_TAGS)))
	GO_VET := go tool vet \
		-atomic \
		-bool \
		-copylocks \
		-nilfunc \
		-printf \
		-shadow \
		-rangeloops \
		-unreachable \
		-unsafeptr \
		-unusedresult \
		.
else
	GO_VET := @echo "go vet skipped -- not supported on this version of Go"
endif

fmt: ## fmt
	@echo gofmt -l
	@OUTPUT=`gofmt -l . 2>&1`; \
	if [ "$$OUTPUT" ]; then \
		echo "gofmt must be run on the following files:"; \
        echo "$$OUTPUT"; \
        exit 1; \
    fi

lint:
	@golangci-lint run
.PHONY: lint

vet: ## vet
	$(GO_VET)

default: fmt lint vet

local: clean ## 构建二进制
	@echo "build sealer and sealutil bin"
	hack/build.sh

clean: ## clean
	@rm -rf _output/bin
