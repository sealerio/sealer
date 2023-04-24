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
#
# ==============================================================================
# wget https://github.com/google/addlicense/releases/download/v1.0.0/addlicense_1.0.0_Linux_x86_64.tar.gz
# Makefile helper functions for copyright
#

# filelicense: SHELL:=/bin/bash
# ## filelicense: add license for all files
# filelicense:
# 	for file in ${Dirs} ; do \
# 		if [[  $$file != '_output' && $$file != 'docs' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' ]]; then \
# 			$(ADDLICENSE_BIN)  -y $(shell date +"%Y") -c "Alibaba Group Holding Ltd." -f scripts/LICENSE_TEMPLATE ./$$file ; \
# 		fi \
#     done
#
# install-addlicense:
# ifeq (, $(shell which addlicense))
# 	@{ \
# 	set -e ;\
# 	LICENSE_TMP_DIR=$$(mktemp -d) ;\
# 	cd $$LICENSE_TMP_DIR ;\
# 	go mod init tmp ;\
# 	go get -v github.com/google/addlicense ;\
# 	rm -rf $$LICENSE_TMP_DIR ;\
# 	}
# ADDLICENSE_BIN=$(GOBIN)/addlicense
# else
# ADDLICENSE_BIN=$(shell which addlicense)
# endif
#
# ## install-gosec: check license if not exist install addlicense tools.
# install-gosec:
# ifeq (, $(shell which gosec))
# 	@{ \
# 	set -e ;\
# 	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOBIN) v2.2.0 ;\
# 	}
# GOSEC_BIN=$(GOBIN)/gosec
# else
# GOSEC_BIN=$(shell which gosec)
# endif
# gosec: install-gosec
# 	$(GOSEC_BIN) ./...

LICENSE_TEMPLATE ?= $(ROOT_DIR)/scripts/LICENSE_TEMPLATE

# TODO: GOBIN -> TOOLS_DIR
## copyright.verify: Validate boilerplate headers for assign files
.PHONY: copyright.verify
copyright.verify: tools.verify.addlicense
	@echo "===========> Validate boilerplate headers for assign files starting in the $(ROOT_DIR) directory"
	@$(GOBIN)/addlicense -v -check -ignore **/test/** -f $(LICENSE_TEMPLATE) $(CODE_DIRS)
	@echo "===========> End of boilerplate headers check..."

## copyright.add: Add the boilerplate headers for all files
.PHONY: copyright.add
copyright.add: tools.verify.addlicense
	@echo "===========> Adding $(LICENSE_TEMPLATE) the boilerplate headers for all files"
	@$(GOBIN)/addlicense -y $(shell date +"%Y") -v -c "Alibaba Group Holding Ltd." -f $(LICENSE_TEMPLATE) $(CODE_DIRS)
	@echo "===========> End the copyright is added..."

# Addlicense Flags:
#   -c string
#         copyright holder (default "Google LLC")
#   -check
#         check only mode: verify presence of license headers and exit with non-zero code if missing
#   -f string
#         license file
#   -ignore value
#         file patterns to ignore, for example: -ignore **/*.go -ignore vendor/**
#   -l string
#         license type: apache, bsd, mit, mpl (default "apache")
#   -s    Include SPDX identifier in license header. Set -s=only to only include SPDX identifier.
#   -skip value
#         [deprecated: see -ignore] file extensions to skip, for example: -skip rb -skip go
#   -v    verbose mode: print the name of the files that are modified or were skipped
#   -y string
#         copyright year(s) (default "2023")

## copyright.help: Show copyright help
.PHONY: copyright.help
copyright.help: scripts/make-rules/copyright.mk
	$(call smallhelp)