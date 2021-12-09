// Copyright © 2021 Alibaba Group Holding Ltd.
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

package save

import (
	"context"
	"fmt"

	"github.com/alibaba/sealer/logger"
	distribution "github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/reference"
	"github.com/distribution/distribution/v3/registry/proxy"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	"github.com/opencontainers/go-digest"
)

const (
	proxyURL      = `https://registry-1.docker.io`
	configFileSys = `filesystem`
	configRootDir = `rootdirectory`
)

func (ls *DefaultImageSaver) SaveImages(images []string, dir, arch string) error {
	logger.Info("Saving images from %s into %s...", proxyURL, dir)
	registry, err := NewProxyRegistry(ls.ctx, dir)
	if err != nil {
		return fmt.Errorf("init registry error: %v", err)
	}
	for _, image := range images {
		err := ls.save(image, arch, registry)
		if err != nil {
			return fmt.Errorf("save image %s error: %v", image, err)
		}
	}
	return nil
}

func (ls *DefaultImageSaver) save(image string, arch string, registry distribution.Namespace) error {
	named, err := parseNormalizedNamed(image)
	if err != nil {
		return fmt.Errorf("parse image name error: %v", err)
	}

	repo, err := ls.getRepository(named, registry)
	if err != nil {
		return err
	}

	imageDigest, err := ls.saveManifestAndGetDigest(named, repo, arch)
	if err != nil {
		return err
	}

	err = ls.saveBlobs(named, repo, imageDigest)
	if err != nil {
		return err
	}

	return nil
}

func NewProxyRegistry(ctx context.Context, rootdir string) (distribution.Namespace, error) {
	config := configuration.Configuration{
		Proxy: configuration.Proxy{
			RemoteURL: proxyURL,
		},
		Storage: configuration.Storage{
			configFileSys: configuration.Parameters{configRootDir: rootdir},
		},
	}
	driver, err := factory.Create(config.Storage.Type(), config.Storage.Parameters())
	if err != nil {
		return nil, fmt.Errorf("create storage driver error: %v", err)
	}
	registry, err := storage.NewRegistry(ctx, driver, make([]storage.RegistryOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("create local registry error: %v", err)
	}

	proxy, err := proxy.NewRegistryPullThroughCache(ctx, registry, driver, config.Proxy)
	if err != nil {
		return nil, fmt.Errorf("create proxy registry error: %v", err)
	}
	return proxy, nil
}

func (ls *DefaultImageSaver) getRepository(named Named, registry distribution.Namespace) (distribution.Repository, error) {
	repoName, err := reference.WithName(named.Repo())
	if err != nil {
		return nil, fmt.Errorf("get repository name error: %v", err)
	}
	logger.Info("Saving image %s/%s:%s...\n", named.domain, named.repo, named.tag)
	repo, err := registry.Repository(ls.ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("get repository error: %v", err)
	}
	return repo, nil
}

func (ls *DefaultImageSaver) saveManifestAndGetDigest(named Named, repo distribution.Repository, arch string) (digest.Digest, error) {
	manifest, err := repo.Manifests(ls.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return digest.Digest(""), fmt.Errorf("get manifest service error: %v", err)
	}

	desc, err := repo.Tags(ls.ctx).Get(ls.ctx, named.tag)
	if err != nil {
		return digest.Digest(""), fmt.Errorf("get tag descriptor error: %v", err)
	}

	manifestListJSON, err := manifest.Get(ls.ctx, desc.Digest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return digest.Digest(""), fmt.Errorf("get image manifest list error: %v", err)
	}

	imageDigest, err := getImageManifestDigest(manifestListJSON, arch)
	if err != nil {
		return digest.Digest(""), fmt.Errorf("get digest error: %v", err)
	}

	return imageDigest, nil
}

func (ls *DefaultImageSaver) saveBlobs(named Named, repo distribution.Repository, imageDigest digest.Digest) error {
	manifest, err := repo.Manifests(ls.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get blob service error: %v", err)
	}
	blobListJSON, err := manifest.Get(ls.ctx, imageDigest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get blob manifest error: %v", err)
	}
	blobList, err := getBlobList(blobListJSON)
	if err != nil {
		return fmt.Errorf("get blob list error: %v", err)
	}
	blobStore := repo.Blobs(ls.ctx)
	for _, blob := range blobList {
		_, err = blobStore.Get(ls.ctx, blob)
		if err != nil {
			return fmt.Errorf("get blob %s error: %v", blob, err)
		}
	}
	logger.Info("Successfully saved image %s/%s:%s\n", named.domain, named.repo, named.tag)
	return nil
}
