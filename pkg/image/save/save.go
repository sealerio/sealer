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
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/reference"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/opencontainers/go-digest"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/docker/auth"
	"github.com/sealerio/sealer/pkg/image/save/distributionpkg/proxy"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	HTTPS               = "https://"
	HTTP                = "http://"
	defaultProxyURL     = "https://registry-1.docker.io"
	configRootDir       = "rootdirectory"
	maxPullGoroutineNum = 2
	maxRetryTime        = 3

	manifestV2       = "application/vnd.docker.distribution.manifest.v2+json"
	manifestOCI      = "application/vnd.oci.image.manifest.v1+json"
	manifestList     = "application/vnd.docker.distribution.manifest.list.v2+json"
	manifestOCIIndex = "application/vnd.oci.image.index.v1+json"
)

func (is *DefaultImageSaver) SaveImages(images []string, dir string, platform v1.Platform) error {
	//init a pipe for display pull message
	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	is.progressOut = streamformatter.NewJSONProgressOutput(writer, false)

	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, dockerstreams.NewOut(common.StdOut), nil)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Warnf("error occurs in display progressing, err: %s", err)
		}
	}()

	existFlag := make(map[string]struct{})
	//handle image name
	for _, image := range images {
		named, err := ParseNormalizedNamed(image, "")
		if err != nil {
			return fmt.Errorf("failed to parse image name:: %v", err)
		}

		//check if image is duplicate
		if _, exist := existFlag[named.FullName()]; exist {
			continue
		} else {
			existFlag[named.FullName()] = struct{}{}
		}

		//check if image exist in disk
		if err := is.isImageExist(named, dir, platform); err == nil {
			continue
		}
		is.domainToImages[named.domain+named.repo] = append(is.domainToImages[named.domain+named.repo], named)
		progress.Message(is.progressOut, "", fmt.Sprintf("Pulling image: %s", named.FullName()))
	}

	//perform image save ability
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	for _, nameds := range is.domainToImages {
		tmpnameds := nameds
		numCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()
			registry, err := NewProxyRegistry(is.ctx, dir, tmpnameds[0].domain)
			if err != nil {
				return fmt.Errorf("failed to init registry: %v", err)
			}
			err = is.save(tmpnameds, platform, registry)
			if err != nil {
				return fmt.Errorf("failed to save domain %s image: %v", tmpnameds[0].domain, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if len(images) != 0 {
		progress.Message(is.progressOut, "", "Status: images save success")
	}
	return nil
}

// isImageExist check if an image exist in local
func (is *DefaultImageSaver) isImageExist(named Named, dir string, platform v1.Platform) error {
	config := configuration.Configuration{
		Storage: configuration.Storage{
			driverName: configuration.Parameters{configRootDir: dir},
		},
	}
	registry, err := newRegistry(is.ctx, config)
	if err != nil {
		return err
	}

	repo, err := is.getRepository(named, registry)
	if err != nil {
		return err
	}

	blobList, err := is.getLocalDigest(named, repo, platform)
	if err != nil {
		return err
	}

	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)
	for _, blob := range blobList {
		numCh <- struct{}{}
		tmpblob := blob
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()
			_, err := registry.BlobStatter().Stat(is.ctx, tmpblob)
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// newRegistry init a local registry service
func newRegistry(ctx context.Context, config configuration.Configuration) (distribution.Namespace, error) {
	driver, err := factory.Create(config.Storage.Type(), config.Storage.Parameters())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage driver: %v", err)
	}

	//create a local registry service
	registry, err := storage.NewRegistry(ctx, driver, make([]storage.RegistryOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create local registry: %v", err)
	}
	return registry, nil
}

// getLocalDigest get local image digest list
func (is *DefaultImageSaver) getLocalDigest(named Named, repo distribution.Repository, platform v1.Platform) ([]digest.Digest, error) {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest service: %v", err)
	}

	tagService := repo.Tags(is.ctx)
	desc, err := tagService.Get(is.ctx, named.Tag())
	if err != nil {
		return nil, fmt.Errorf("failed to get %s tag descriptor in local: %v", named.repo, err)
	}

	imageDigest, err := is.handleManifest(manifest, desc.Digest, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get digest: %v", err)
	}

	blobListJSON, err := manifest.Get(is.ctx, imageDigest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return nil, err
	}

	blobList, err := getBlobList(blobListJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob list: %v", err)
	}
	return blobList, nil
}

func (is *DefaultImageSaver) SaveImagesWithAuth(imageList ImageListWithAuth, dir string, platform v1.Platform) error {
	//init a pipe for display pull message
	reader, writer := io.Pipe()
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()
	is.progressOut = streamformatter.NewJSONProgressOutput(writer, false)
	is.ctx = context.Background()
	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, dockerstreams.NewOut(common.StdOut), nil)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Warnf("error occurs in display progressing, err: %s", err)
		}
	}()

	//perform image save ability
	eg, _ := errgroup.WithContext(context.Background())
	numCh := make(chan struct{}, maxPullGoroutineNum)

	//handle imageList
	for _, section := range imageList {
		for _, nameds := range section.Images {
			tmpnameds := nameds
			progress.Message(is.progressOut, "", fmt.Sprintf("Pulling image: %s", tmpnameds[0].FullName()))
			numCh <- struct{}{}
			eg.Go(func() error {
				defer func() {
					<-numCh
				}()
				if err := is.download(dir, platform, section, tmpnameds, maxRetryTime); err != nil {
					return err
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}

	if len(imageList) != 0 {
		progress.Message(is.progressOut, "", "Status: images save success")
	}
	return nil
}

func (is *DefaultImageSaver) download(dir string, platform v1.Platform, section Section, nameds []Named, retryTime int) error {
	registry, err := NewProxyRegistryWithAuth(is.ctx, section.Username, section.Password, dir, nameds[0].domain)
	if err != nil {
		return fmt.Errorf("failed to init registry: %v", err)
	}
	err = is.save(nameds, platform, registry)
	if err != nil {
		return fmt.Errorf("failed to save domain %s image: %v", nameds[0], err)
	}

	// double check whether the image is unbroken
	var imageExistError error
	var imageExistErrorNamed Named
	for _, named := range nameds {
		imageExistError = is.isImageExist(named, dir, platform)
		if imageExistError != nil {
			imageExistErrorNamed = named
			break
		}
	}
	if imageExistError == nil {
		return nil
	}
	if retryTime <= 0 {
		return imageExistError
	}
	// retry to download
	progress.Message(is.progressOut, "", fmt.Sprintf("Retry: failed to save image(%s) and retry it", imageExistErrorNamed.FullName()))
	return is.download(dir, platform, section, nameds, retryTime-1)
}

// TODO: support retry mechanism here
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
		return nil, fmt.Errorf("failed to get repository name: %v", err)
	}
	repo, err := registry.Repository(is.ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %v", err)
	}
	return repo, nil
}

func (is *DefaultImageSaver) saveManifestAndGetDigest(nameds []Named, repo distribution.Repository, platform v1.Platform) ([]digest.Digest, error) {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest service: %v", err)
	}

	var (
		// lock protects imageDigests
		lock         sync.Mutex
		imageDigests = make([]digest.Digest, 0)
		numCh        = make(chan struct{}, maxPullGoroutineNum)
	)

	eg, _ := errgroup.WithContext(context.Background())

	for _, named := range nameds {
		tmpnamed := named
		numCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()

			desc, err := repo.Tags(is.ctx).Get(is.ctx, tmpnamed.tag)
			if err != nil {
				return fmt.Errorf("failed to get %s tag descriptor: %v. Try \"docker login\" if you are using a private registry", tmpnamed.repo, err)
			}
			imageDigest, err := is.handleManifest(manifest, desc.Digest, platform)
			if err != nil {
				return fmt.Errorf("failed to get digest: %v", err)
			}

			lock.Lock()
			defer lock.Unlock()
			imageDigests = append(imageDigests, imageDigest)
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
		return "", fmt.Errorf("failed to get image manifest: %v", err)
	}
	ct, p, err := mani.Payload()
	if err != nil {
		return "", fmt.Errorf("failed to get image manifest payload: %v", err)
	}
	switch ct {
	case manifestV2, manifestOCI:
		return imagedigest, nil
	case manifestList, manifestOCIIndex:
		imageDigest, err := getImageManifestDigest(p, platform)
		if err != nil {
			return "", fmt.Errorf("failed to get digest from manifest list: %v", err)
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
		return "", fmt.Errorf("unrecognized manifest content type")
	}
}

func (is *DefaultImageSaver) saveBlobs(imageDigests []digest.Digest, repo distribution.Repository) error {
	manifest, err := repo.Manifests(is.ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("failed to get blob service: %v", err)
	}

	var (
		// lock protects blobLists
		lock      sync.Mutex
		blobLists = make([]digest.Digest, 0)
		numCh     = make(chan struct{}, maxPullGoroutineNum)
	)

	eg, _ := errgroup.WithContext(context.Background())

	//get blob list
	//each blob identified by a digest
	for _, imageDigest := range imageDigests {
		tmpImageDigest := imageDigest
		numCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()

			blobListJSON, err := manifest.Get(is.ctx, tmpImageDigest, make([]distribution.ManifestServiceOption, 0)...)
			if err != nil {
				return err
			}

			blobList, err := getBlobList(blobListJSON)
			if err != nil {
				return fmt.Errorf("failed to get blob list: %v", err)
			}

			lock.Lock()
			defer lock.Unlock()
			blobLists = append(blobLists, blobList...)
			return nil
		})
	}
	if err = eg.Wait(); err != nil {
		return err
	}

	//pull and save each blob
	blobStore := repo.Blobs(is.ctx)
	for _, blob := range blobLists {
		tmpBlob := blob
		numCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-numCh
			}()

			if len(string(tmpBlob)) < 19 {
				return nil
			}
			simpleDgst := string(tmpBlob)[7:19]

			_, err = blobStore.Stat(is.ctx, tmpBlob)
			if err == nil { //blob already exist
				progress.Update(is.progressOut, simpleDgst, "already exists")
				return nil
			}
			reader, err := blobStore.Open(is.ctx, tmpBlob)
			if err != nil {
				return fmt.Errorf("failed to get blob %s: %v", tmpBlob, err)
			}

			size, err := reader.Seek(0, io.SeekEnd)
			if err != nil {
				return fmt.Errorf("seek end error when save blob %s: %v", tmpBlob, err)
			}
			_, err = reader.Seek(0, io.SeekStart)
			if err != nil {
				return fmt.Errorf("failed to seek start when save blob %s: %v", tmpBlob, err)
			}
			preader := progress.NewProgressReader(reader, is.progressOut, size, simpleDgst, "Downloading")

			defer func() {
				_ = reader.Close()
				_ = preader.Close()
				progress.Update(is.progressOut, simpleDgst, "Download complete")
			}()

			//store to local filesystem
			//content, err := ioutil.ReadAll(preader)
			bf := bufio.NewReader(preader)
			if err != nil {
				return fmt.Errorf("blob %s content error: %v", tmpBlob, err)
			}
			bw, err := blobStore.Create(is.ctx)
			if err != nil {
				return fmt.Errorf("failed to create blob store writer: %v", err)
			}
			if _, err = bf.WriteTo(bw); err != nil {
				return fmt.Errorf("failed to write blob to service: %v", err)
			}
			_, err = bw.Commit(is.ctx, distribution.Descriptor{
				MediaType: "",
				Size:      bw.Size(),
				Digest:    tmpBlob,
			})
			if err != nil {
				return fmt.Errorf("failed to store blob %s to local: %v", tmpBlob, err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func NewProxyRegistryWithAuth(ctx context.Context, username, password, rootdir, domain string) (distribution.Namespace, error) {
	// set the URL of registry
	proxyURL := HTTPS + domain
	if domain == defaultDomain {
		proxyURL = defaultProxyURL
	}

	config := configuration.Configuration{
		Proxy: configuration.Proxy{
			RemoteURL: proxyURL,
			Username:  username,
			Password:  password,
		},
		Storage: configuration.Storage{
			driverName: configuration.Parameters{configRootDir: rootdir},
		},
	}
	return newProxyRegistry(ctx, config)
}

func NewProxyRegistry(ctx context.Context, rootdir, domain string) (distribution.Namespace, error) {
	// set the URL of registry
	proxyURL := HTTPS + domain
	if domain == defaultDomain {
		proxyURL = defaultProxyURL
	}

	svc, err := auth.NewDockerAuthService()
	if err != nil {
		return nil, fmt.Errorf("failed to read default auth file: %v", err)
	}
	defaultAuth := types.AuthConfig{ServerAddress: domain}
	authConfig, err := svc.GetAuthByDomain(domain)
	//ignore err when is there is no username and password.
	//regard it as a public registry
	//only report parse error
	if err != nil && authConfig != defaultAuth {
		return nil, fmt.Errorf("failed to get authentication info: %v", err)
	}

	config := configuration.Configuration{
		Proxy: configuration.Proxy{
			RemoteURL: proxyURL,
			Username:  authConfig.Username,
			Password:  authConfig.Password,
		},
		Storage: configuration.Storage{
			driverName: configuration.Parameters{configRootDir: rootdir},
		},
	}

	return newProxyRegistry(ctx, config)
}

func newProxyRegistry(ctx context.Context, config configuration.Configuration) (distribution.Namespace, error) {
	driver, err := factory.Create(config.Storage.Type(), config.Storage.Parameters())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage driver: %v", err)
	}

	//create a local registry service
	registry, err := storage.NewRegistry(ctx, driver, make([]storage.RegistryOption, 0)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create local registry: %v", err)
	}

	proxyRegistry, err := proxy.NewRegistryPullThroughCache(ctx, registry, driver, config.Proxy)
	if err != nil { // try http
		logrus.Warnf("https error: %v, sealer try to use http", err)
		config.Proxy.RemoteURL = strings.Replace(config.Proxy.RemoteURL, HTTPS, HTTP, 1)
		proxyRegistry, err = proxy.NewRegistryPullThroughCache(ctx, registry, driver, config.Proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy registry: %v", err)
		}
	}
	return proxyRegistry, nil
}
