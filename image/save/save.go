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
	"sync"

	distribution "github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/reference"
	"github.com/distribution/distribution/v3/registry/proxy"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	urlPrefix           = "https://"
	defauleProxyURL     = "https://registry-1.docker.io"
	configRootDir       = "rootdirectory"
	maxPullGoroutineNum = 10
)

func (is *DefaultImageSaver) SaveImages(images []string, dir string, platform v1.Platform) error {
	for _, image := range images {
		named, err := parseNormalizedNamed(image)
		if err != nil {
			return fmt.Errorf("parse image name error: %v", err)
		}
		is.domainToImages[named.domain+named.repo] = append(is.domainToImages[named.domain+named.repo], named)
	}
	for _, nameds := range is.domainToImages {
		registry, err := NewProxyRegistry(is.ctx, dir, nameds[0].domain)
		if err != nil {
			return fmt.Errorf("init registry error: %v", err)
		}
		err = is.save(nameds, platform, registry)
		if err != nil {
			return fmt.Errorf("save domain %s image error: %v", nameds[0].domain, err)
		}
	}
	return nil
}

func (is *DefaultImageSaver) save(nameds []Named, platform v1.Platform, registry distribution.Namespace) error {
	repo, err := is.getRepository(nameds[0], registry)
	if err != nil {
		return err
	}

	imageDigests, err := is.saveManifestAndGetDigest(nameds, repo, platform)
	if err != nil {
		return err
	}

	err = is.saveBlobs(imageDigests, repo)
	if err != nil {
		return err
	}

	return nil
}

func NewProxyRegistry(ctx context.Context, rootdir, domain string) (distribution.Namespace, error) {
	// set the URL of registry
	proxyURL := urlPrefix + domain
	if domain == defaultDomain {
		proxyURL = defauleProxyURL
	}

	config := configuration.Configuration{
		Proxy: configuration.Proxy{
			RemoteURL: proxyURL,
		},
		Storage: configuration.Storage{
			driverName: configuration.Parameters{configRootDir: rootdir},
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

func (is *DefaultImageSaver) getRepository(named Named, registry distribution.Namespace) (distribution.Repository, error) {
	repoName, err := reference.WithName(named.Repo())
	if err != nil {
		return nil, fmt.Errorf("get repository name error: %v", err)
	}
	repo, err := registry.Repository(is.ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("get repository error: %v", err)
	}
	return repo, nil
}

func (is *DefaultImageSaver) saveManifestAndGetDigest(nameds []Named, repo distribution.Repository, platform v1.Platform) ([]digest.Digest, error) {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("get manifest service error: %v", err)
	}
	digestCh := make(chan digest.Digest)
	numCh := make(chan bool, maxPullGoroutineNum)
	errCh := make(chan error)
	imageDigests := make([]digest.Digest, 0)
	for _, named := range nameds {
		tmpnamed := named
		numCh <- true
		go func(named Named) {
			desc, err := repo.Tags(is.ctx).Get(is.ctx, named.tag)
			if err != nil {
				errCh <- fmt.Errorf("get %s tag descriptor error: %v", named.repo, err)
			}

			manifestListJSON, err := manifest.Get(is.ctx, desc.Digest, make([]distribution.ManifestServiceOption, 0)...)
			if err != nil {
				errCh <- fmt.Errorf("get image manifest list error: %v", err)
			}

			imageDigest, err := getImageManifestDigest(manifestListJSON, platform)
			if err != nil {
				errCh <- fmt.Errorf("get digest error: %v", err)
			}
			digestCh <- imageDigest
			<-numCh
		}(tmpnamed)
	}
	for range nameds {
		select {
		case imageDigest := <-digestCh:
			imageDigests = append(imageDigests, imageDigest)
		case err = <-errCh:
			return nil, err
		}
	}

	return imageDigests, nil
}

func (is *DefaultImageSaver) saveBlobs(imageDigests []digest.Digest, repo distribution.Repository) error {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get blob service error: %v", err)
	}
	blobDigestCh := make(chan []digest.Digest)
	errCh := make(chan error)
	numCh := make(chan bool, maxPullGoroutineNum)
	blobList := make([]digest.Digest, 0)
	blobMap := make(map[digest.Digest]bool)
	for _, imageDigest := range imageDigests {
		tmpImageDigest := imageDigest
		numCh <- true
		go func(digest digest.Digest) {
			blobListJSON, err := manifest.Get(is.ctx, digest, make([]distribution.ManifestServiceOption, 0)...)
			if err != nil {
				errCh <- fmt.Errorf("get blob manifest error: %v", err)
			}
			blobList, err := getBlobList(blobListJSON)
			if err != nil {
				errCh <- fmt.Errorf("get blob list error: %v", err)
			}
			blobDigestCh <- blobList
			<-numCh
		}(tmpImageDigest)
	}
	for range imageDigests {
		select {
		case blobDigest := <-blobDigestCh:
			for _, digest := range blobDigest {
				if !blobMap[digest] {
					blobList = append(blobList, digest)
					blobMap[digest] = true
				}
			}
		case err = <-errCh:
			return err
		}
	}
	var wg = sync.WaitGroup{}
	blobStore := repo.Blobs(is.ctx)
	for _, blob := range blobList {
		tmpBlob := blob
		numCh <- true
		wg.Add(1)
		go func(blob digest.Digest) {
			_, err = blobStore.Get(is.ctx, blob)
			if err != nil {
				errCh <- fmt.Errorf("get blob %s error: %v", blob, err)
			}
			<-numCh
			wg.Done()
		}(tmpBlob)
	}
	wg.Wait()
	return nil
}
