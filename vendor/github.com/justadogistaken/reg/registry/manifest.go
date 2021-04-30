package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

var (
	// ErrUnexpectedSchemaVersion a specific schema version was requested, but was not returned
	ErrUnexpectedSchemaVersion = errors.New("recieved a different schema version than expected")
)

// Manifest returns the manifest for a specific repository:tag.
func (registry *Registry) Manifest(ctx context.Context, repository, ref string) (distribution.Manifest, error) {
	uri := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("registry.manifests uri=%s repository=%s ref=%s", uri, repository, ref)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", schema2.MediaTypeManifest)

	resp, err := registry.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	registry.Logf("registry.manifests resp.Status=%s, body=%s", resp.Status, body)

	m, _, err := distribution.UnmarshalManifest(resp.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// ManifestList gets the registry v2 manifest list.
func (registry *Registry) ManifestList(ctx context.Context, repository, ref string) (manifestlist.ManifestList, error) {
	uri := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("registry.manifests uri=%s repository=%s ref=%s", uri, repository, ref)

	var m manifestlist.ManifestList
	if _, err := registry.getJSON(ctx, uri, &m); err != nil {
		registry.Logf("registry.manifests response=%v", m)
		return m, err
	}

	return m, nil
}

// ManifestV2 gets the registry v2 manifest.
func (registry *Registry) ManifestV2(ctx context.Context, repository, ref string) (schema2.Manifest, error) {
	uri := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("registry.manifests uri=%s repository=%s ref=%s", uri, repository, ref)

	var m schema2.Manifest
	if _, err := registry.getJSON(ctx, uri, &m); err != nil {
		registry.Logf("registry.manifests response=%v", m)
		return m, err
	}

	if m.Versioned.SchemaVersion != 2 {
		return m, ErrUnexpectedSchemaVersion
	}

	return m, nil
}

// ManifestV1 gets the registry v1 manifest.
func (registry *Registry) ManifestV1(ctx context.Context, repository, ref string) (schema1.SignedManifest, error) {
	uri := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("registry.manifests uri=%s repository=%s ref=%s", uri, repository, ref)

	var m schema1.SignedManifest
	if _, err := registry.getJSON(ctx, uri, &m); err != nil {
		registry.Logf("registry.manifests response=%v", m)
		return m, err
	}

	if m.Versioned.SchemaVersion != 1 {
		return m, ErrUnexpectedSchemaVersion
	}

	return m, nil
}

func (registry *Registry) initManifestsPut(repository, ref string) (string, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("for getting token, registry.manifest.put url=%s repository=%s reference=%s", url, repository, ref)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := registry.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	token := resp.Header.Get("Request-Token")
	return token, nil
}

// PutManifest calls a PUT for the specific manifest for an image.
func (registry *Registry) PutManifest(ctx context.Context, repository, ref string, manifest distribution.Manifest) error {
	token, err := registry.initManifestsPut(repository, ref)
	if err != nil {
		return err
	}

	url := registry.url("/v2/%s/manifests/%s", repository, ref)
	registry.Logf("registry.manifest.put url=%s repository=%s reference=%s", url, repository, ref)

	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", schema2.MediaTypeManifest)
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else {
		registry.Logf("token is empty, you may ignore this")
	}

	resp, err := registry.Client.Do(req.WithContext(ctx))
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
