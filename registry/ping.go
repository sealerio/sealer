package registry

import (
	"context"
	"net/http"
	"strings"
)

// Pingable checks pingable
func (registry *Registry) Pingable() bool {
	// Currently *.gcr.io/v2 can't be ping if users have each projects auth
	return !strings.HasSuffix(registry.URL, "gcr.io")
}

// Ping tries to contact a registry URL to make sure it is up and accessible.
func (registry *Registry) Ping(ctx context.Context) error {
	url := registry.url("/v2/")
	registry.Logf("registry.ping url=%s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := registry.Client.Do(req.WithContext(ctx))
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
