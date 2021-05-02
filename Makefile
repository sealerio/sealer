GitTag=$(version)

build: clean ## 构建二进制
	@echo "build sealer and sealutil bin"
	hack/build.sh $(GitTag)

clean: ## clean
	@rm -rf _output
