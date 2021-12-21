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

	"github.com/alibaba/sealer/utils"
	distribution "github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/reference"
	"github.com/distribution/distribution/v3/registry/proxy"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	"github.com/docker/docker/api/types"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
)

const (
	urlPrefix           = "https://"
	defauleProxyURL     = "https://registry-1.docker.io"
	configRootDir       = "rootdirectory"
	maxPullGoroutineNum = 10

	manifestV2       = "application/vnd.docker.distribution.manifest.v2+json"
	manifestOCI      = "application/vnd.oci.image.manifest.v1+json"
	manifestList     = "application/vnd.docker.distribution.manifest.list.v2+json"
	manifestOCIIndex = "application/vnd.oci.image.index.v1+json"
)

func (is *DefaultImageSaver) SaveImages(images []string, dir string, platform v1.Platform) error {
	for _, image := range images {
		named, err := parseNormalizedNamed(image)
		if err != nil {
			return fmt.Errorf("parse image name error: %v", err)
		}
		is.domainToImages[named.domain+named.repo] = append(is.domainToImages[named.domain+named.repo], named)
	}
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	for _, nameds := range is.domainToImages {
		tmpnameds := nameds
		numCh <- struct{}{}
		eg.Go(func() error {
			registry, err := NewProxyRegistry(is.ctx, dir, tmpnameds[0].domain)
			if err != nil {
				return fmt.Errorf("init registry error: %v", err)
			}
			err = is.save(tmpnameds, platform, registry)
			if err != nil {
				return fmt.Errorf("save domain %s image error: %v", tmpnameds[0].domain, err)
			}
			<-numCh
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
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

	var defaultAuth = types.AuthConfig{ServerAddress: domain}
	auth, err := utils.GetDockerAuthInfoFromDocker(domain)
	//ignore err when is there is no username and password.
	//regard it as a public registry
	//only report parse error
	if err != nil && auth != defaultAuth {
		return nil, fmt.Errorf("get authentication info error: %v", err)
	}
	config := configuration.Configuration{
		Proxy: configuration.Proxy{
			RemoteURL: proxyURL,
			Username:  auth.Username,
			Password:  auth.Password,
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
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	imageDigests := make([]digest.Digest, 0)
	for _, named := range nameds {
		tmpnamed := named
		numCh <- struct{}{}
		eg.Go(func() error {
			desc, err := repo.Tags(is.ctx).Get(is.ctx, tmpnamed.tag)
			if err != nil {
				return fmt.Errorf("get %s tag descriptor error: %v, try \"docker login\" if you are using a private registry", tmpnamed.repo, err)
			}
			imageDigest, err := is.handleManifest(manifest, desc.Digest, platform)
			if err != nil {
				return fmt.Errorf("get digest error: %v", err)
			}
			imageDigests = append(imageDigests, imageDigest)
			<-numCh
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return imageDigests, nil
}

func (is *DefaultImageSaver) handleManifest(manifest distribution.ManifestService, imagedigest digest.Digest, platform v1.Platform) (digest.Digest, error) {
	mani, err := manifest.Get(is.ctx, imagedigest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return digest.Digest(""), fmt.Errorf("get image manifest error: %v", err)
	}
	ct, p, err := mani.Payload()
	if err != nil {
		return digest.Digest(""), fmt.Errorf("failed to get image manifest payload: %v", err)
	}
	switch ct {
	case manifestV2, manifestOCI:
		return imagedigest, nil
	case manifestList, manifestOCIIndex:
		imageDigest, err := getImageManifestDigest(p, platform)
		if err != nil {
			return digest.Digest(""), fmt.Errorf("get digest from manifest list error: %v", err)
		}
		return imageDigest, nil
	case "":
		//OCI image or image index - no media type in the content

		//First see if it is a list
		imageDigest, _ := getImageManifestDigest(p, platform)
		if imageDigest != "" {
			return imageDigest, nil
		}
		//If not list, then assume it must be an image manifest
		return imagedigest, nil
	default:
		return digest.Digest(""), fmt.Errorf("unrecognized manifest content type")
	}
}

func (is *DefaultImageSaver) saveBlobs(imageDigests []digest.Digest, repo distribution.Repository) error {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get blob service error: %v", err)
	}
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	blobLists := make([]digest.Digest, 0)
	for _, imageDigest := range imageDigests {
		tmpImageDigest := imageDigest
		numCh <- struct{}{}
		eg.Go(func() error {
			blobListJSON, err := manifest.Get(is.ctx, tmpImageDigest, make([]distribution.ManifestServiceOption, 0)...)
			if err != nil {
				return err
			}

			blobList, err := getBlobList(blobListJSON)
			if err != nil {
				return fmt.Errorf("get blob list error: %v", err)
			}
			blobLists = append(blobLists, blobList...)
			<-numCh
			return nil
		})
	}
	if err = eg.Wait(); err != nil {
		return err
	}

	blobStore := repo.Blobs(is.ctx)
	for _, blob := range blobLists {
		tmpBlob := blob
		numCh <- struct{}{}
		eg.Go(func() error {
			_, err = blobStore.Get(is.ctx, tmpBlob)
			if err != nil {
				return fmt.Errorf("get blob %s error: %v", tmpBlob, err)
			}
			<-numCh
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
