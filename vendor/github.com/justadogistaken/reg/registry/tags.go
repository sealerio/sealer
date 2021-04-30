package registry

import "context"

type tagsResponse struct {
	Tags []string `json:"tags"`
}

// Tags returns the tags for a specific repository.
func (registry *Registry) Tags(ctx context.Context, repository string) ([]string, error) {
	url := registry.url("/v2/%s/tags/list", repository)
	registry.Logf("registry.tags url=%s repository=%s", url, repository)

	var response tagsResponse
	if _, err := registry.getJSON(ctx, url, &response); err != nil {
		return nil, err
	}

	return response.Tags, nil
}
