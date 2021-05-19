package registry

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/docker/distribution"

	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
)

// DownloadLayer downloads a specific layer by digest for a repository.
func (registry *Registry) DownloadLayer(ctx context.Context, repository string, digest digest.Digest) (io.ReadCloser, error) {
	url := registry.url("/v2/%s/blobs/%s", repository, digest)
	registry.Logf("registry.layer.download url=%s repository=%s digest=%s", url, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// UploadLayer uploads a specific layer by digest for a repository.
func (registry *Registry) UploadLayer(ctx context.Context, repository string, digs reference.Reference, content io.Reader) error {
	uploadURL, token, err := registry.initiateUpload(ctx, repository)
	if err != nil {
		return err
	}

	q := uploadURL.Query()
	q.Set("digest", digs.String())
	uploadURL.RawQuery = q.Encode()
	dig, ok := digs.(digest.Digest)
	if !ok {
		return errors.New("layer digest is necessary when push a layer")
	}
	// used to verify compressed layer hash
	hash := dig.Algorithm().Hash()
	uploadReader := io.TeeReader(content, hash)
	registry.Logf("registry.layer.upload url=%s repository=%s digest=%s", uploadURL, repository, digs)
	upload, err := http.NewRequest("PUT", uploadURL.String(), uploadReader)
	if err != nil {
		return err
	}

	upload.Header.Set("Content-Type", "application/octet-stream")
	upload.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	_, err = registry.Client.Do(upload.WithContext(ctx))
	currentDigestHex := hex.EncodeToString(hash.Sum(nil))
	if currentDigestHex != dig.Hex() {
		return fmt.Errorf("failed to push layer, err: layer digest changed, original: %s, current: %s", dig.Hex(), currentDigestHex)
	}
	return err
}

// HasLayer returns if the registry contains the specific digest for a repository.
func (registry *Registry) HasLayer(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	checkURL := registry.url("/v2/%s/blobs/%s", repository, digest)
	registry.Logf("registry.layer.check url=%s repository=%s digest=%s", checkURL, repository, digest)

	req, err := http.NewRequest("HEAD", checkURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if err == nil {
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK, nil
	}

	urlErr, ok := err.(*url.Error)
	if !ok {
		return false, err
	}
	httpErr, ok := urlErr.Err.(*httpStatusError)
	if !ok {
		return false, err
	}
	if httpErr.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err
}

// LayerMetadata return the layer metadata by repo and digest
func (registry *Registry) LayerMetadata(repository string, digest digest.Digest) (distribution.Descriptor, error) {
	checkURL := registry.url("/v2/%s/blobs/%s", repository, digest)
	registry.Logf("registry.blob.check url=%s repository=%s digest=%s", checkURL, repository, digest)

	resp, err := registry.Client.Head(checkURL)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return distribution.Descriptor{}, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return distribution.Descriptor{}, errors.New("layer not found")
		}
		return distribution.Descriptor{}, errors.New("unexpected error when retrieving layer")
	}

	return distribution.Descriptor{
		Digest: digest,
		Size:   resp.ContentLength,
	}, nil
}

func (registry *Registry) initiateUpload(ctx context.Context, repository string) (*url.URL, string, error) {
	initiateURL := registry.url("/v2/%s/blobs/uploads/", repository)
	registry.Logf("registry.layer.initiate-upload url=%s repository=%s", initiateURL, repository)

	req, err := http.NewRequest("POST", initiateURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, "", err
	}
	token := resp.Header.Get("Request-Token")
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	locationURL, err := url.Parse(location)
	if err != nil {
		return nil, token, err
	}
	return locationURL, token, nil
}
