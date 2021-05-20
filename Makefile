GitTag=$(version)

build: clean ## build binaries by default
	@echo "build sealer and sealutil bin"
	hack/build.sh $(GitTag)

linux: clean ## build binaries for linux
	@echo "build sealer and sealutil bin for linux"
	GOOS=linux GOARCH=amd64 hack/build.sh $(GitTag)

test-sealer:
	@echo "run e2e test for sealer bin"
	hack/test-sealer.sh

clean: ## clean
	@rm -rf _output
