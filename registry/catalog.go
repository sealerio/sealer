package registry

import (
	"context"
	"net/url"

	"github.com/peterhellberg/link"
)

type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

// Catalog returns the repositories in a registry.
func (registry *Registry) Catalog(ctx context.Context, u string) ([]string, error) {
	if u == "" {
		u = "/v2/_catalog"
	}
	uri := registry.url(u)
	registry.Logf("registry.catalog url=%s", uri)

	var response catalogResponse
	resp, err := registry.getJSON(ctx, uri, &response)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, l := range link.ParseHeader(resp.Header) {
		if l.Rel == "next" {
			unescaped, _ := url.QueryUnescape(l.URI)
			repos, err := registry.Catalog(ctx, unescaped)
			if err != nil {
				return nil, err
			}
			response.Repositories = append(response.Repositories, repos...)
		}
	}

	return response.Repositories, nil
}
