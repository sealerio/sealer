package registry

import (
	"context"
	"testing"

	"github.com/genuinetools/reg/repoutils"
)

func TestDigestFromDockerHub(t *testing.T) {
	ctx := context.Background()
	auth, err := repoutils.GetAuthConfig("", "", "docker.io")
	if err != nil {
		t.Fatalf("Could not get auth config: %s", err)
	}

	r, err := New(ctx, auth, Opt{})
	if err != nil {
		t.Fatalf("Could not create registry instance: %s", err)
	}

	d, err := r.Digest(ctx, Image{Domain: "docker.io", Path: "library/alpine", Tag: "latest"})
	if err != nil {
		t.Fatalf("Could not get digest: %s", err)
	}

	if d == "" {
		t.Error("Empty digest received")
	}
}

func TestDigestFromGCR(t *testing.T) {
	ctx := context.Background()
	auth, err := repoutils.GetAuthConfig("", "", "gcr.io")
	if err != nil {
		t.Fatalf("Could not get auth config: %s", err)
	}

	r, err := New(ctx, auth, Opt{SkipPing: true})
	if err != nil {
		t.Fatalf("Could not create registry instance: %s", err)
	}

	d, err := r.Digest(ctx, Image{Domain: "gcr.io", Path: "google-containers/hyperkube", Tag: "v1.9.9"})
	if err != nil {
		t.Fatalf("Could not get digest: %s", err)
	}

	if d == "" {
		t.Error("Empty digest received")
	}
}
