// Copyright © 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package buildah

import (
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/auth"
)

const (
	policyAbsPath     = "/etc/containers/policy.json"
	registriesAbsPath = "/etc/containers/registries.conf"
	storageConfPath   = "/etc/containers/storage.conf"

	buildahEtcRegistriesConf = `
[registries.search]
registries = ['docker.io']

# Registries that do not use TLS when pulling images or uses self-signed
# certificates.
[registries.insecure]
registries = []

[registries.block]
registries = []
`

	builadhEtcPolicy = `
{
    "default": [
	{
	    "type": "insecureAcceptAnything"
	}
    ],
    "transports":
	{
	    "docker-daemon":
		{
		    "": [{"type":"insecureAcceptAnything"}]
		}
	}
}`

	sealerAuth = `
{
	"auths": {}
}
`

	buildahStorageConf = `
# storage.conf is the configuration file for all tools
# that share the containers/storage libraries
# See man 5 containers-storage.conf for more information
# The "container storage" table contains all of the server options.
[storage]

# Default Storage Driver
driver = "overlay"

# Temporary storage location
runroot = "/var/run/containers/storage"

# Primary Read/Write location of container storage
graphroot = "/var/lib/containers/storage"

[storage.options]
# Storage options to be passed to underlying storage drivers

# Size is used to set a maximum size of the container image.  Only supported by
# certain container storage drivers.
size = ""

[storage.options.thinpool]

# log_level sets the log level of devicemapper.
# 0: LogLevelSuppress 0 (Default)
# 2: LogLevelFatal
# 3: LogLevelErr
# 4: LogLevelWarn
# 5: LogLevelNotice
# 6: LogLevelInfo
# 7: LogLevelDebug
# log_level = "7"`
)

// TODO do we have an util or unified local storage accessing pattern?
func writeFileIfNotExist(path string, content []byte) error {
	_, err := os.Stat(path)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(path), 0750)
		if err != nil {
			return err
		}

		err = os.WriteFile(path, content, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func initBuildah() error {
	if err := writeFileIfNotExist(policyAbsPath, []byte(builadhEtcPolicy)); err != nil {
		return err
	}
	if err := writeFileIfNotExist(registriesAbsPath, []byte(buildahEtcRegistriesConf)); err != nil {
		return err
	}

	storageAbsPath := "/etc/containers/storage.conf"
	if err := writeFileIfNotExist(storageAbsPath, []byte(buildahStorageConf)); err != nil {
		return err
	}

	// TODO maybe this should not be here.
	defaultAuthPath := auth.GetDefaultAuthFilePath()
	if err := writeFileIfNotExist(defaultAuthPath, []byte(sealerAuth)); err != nil {
		return err
	}

	return nil
}
