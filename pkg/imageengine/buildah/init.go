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
	policyAbsPath := "/etc/containers/policy.json"
	if err := writeFileIfNotExist(policyAbsPath, []byte(builadhEtcPolicy)); err != nil {
		return err
	}

	registriesAbsPath := "/etc/containers/registries.conf"
	if err := writeFileIfNotExist(registriesAbsPath, []byte(buildahEtcRegistriesConf)); err != nil {
		return err
	}

	// TODO maybe this should not be here.
	defaultAuthPath := auth.GetDefaultAuthFilePath()
	if err := writeFileIfNotExist(defaultAuthPath, []byte(sealerAuth)); err != nil {
		return err
	}

	return nil
}
