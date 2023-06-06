// Copyright © 2023 Alibaba Group Holding Ltd.
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

package cmd

type ColorMode string

const (
	ColorModeNever  ColorMode = "never"
	ColorModeAlways ColorMode = "always"
)

type LogWriteTo string

const (
	LogWriteToFile   LogWriteTo = "file"
	LogWriteToStdout LogWriteTo = "stdout"
)

type RegistryType string

const (
	RegistryTypeDocker RegistryType = "docker"
	RegistryTypeOci    RegistryType = "oci"
)

const (
	DefaultSealerRootDir = "/var/lib/sealer"
)

func DefaultConfig() *Config {
	return &Config{
		DebugOn:    false,
		LogWriteTo: LogWriteToStdout,
		ColorMode:  ColorModeAlways,
		DataRoot:   DefaultSealerRootDir,
	}
}

type Config struct {
	// Debug refers to the log mode.
	DebugOn bool `json:"debugOn,omitempty"`

	//LogWriteTo where log sealer messages to.
	//default is stdout.
	LogWriteTo LogWriteTo `json:"LogWriteTo,omitempty"`

	//RemoteLoggerURL, if not empty, will send sealer log to this url.
	RemoteLoggerURL string `json:"remoteLoggerURL,omitempty"`

	//RemoteLoggerTaskName which will embedded in the remote logger header, only valid when --remote-logger-url is set
	RemoteLoggerTaskName string `json:"remoteLoggerTaskName,omitempty"`

	//set the log color mode.
	//default is "always",
	ColorMode ColorMode `json:"colorMode,omitempty"`

	//set sealer data root.
	//default is `/var/lib/sealer`
	DataRoot string `json:"dataRoot,omitempty"`

	// Image static related config, such as "image build", "image pull", and so on.
	//Image ImageConfig `json:"image"`

	// Cluster running state related config, such as whether to cache sealer images.
	//Cluster ClusterConfig `json:"cluster"`
}

type ImageConfig struct {
	Build BuildConfig `json:"build"`
}

type BuildConfig struct {
	//docker: use docker registry data format
	//oci: use oci registry data format
	RegistryType RegistryType `json:"registryType,omitempty"`

	// whether to compress registry.
	// default is false.
	Compressed bool `json:"compressed,omitempty"`
}

type ClusterConfig struct {
	//CacheImage: if true, will cache sealer image on remote host with image SHA256.
	//for run: if run the same repeatedly,will skip image distribution when cache is existed.
	//for delete: if true,will not delete remote rootfs files.
	//default is false.
	CacheImage bool `json:"cacheImage,omitempty"`

	// Prune: force delete remote rootfs
	// default is true.
	Prune bool `json:"prune,omitempty"`
}
