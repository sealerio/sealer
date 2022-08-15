package version

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var Timeout = time.Second * 10

// LatestByPrerelease returns the latest released k0s version, if preok is true, prereleases are also accepted.
func LatestByPrerelease(allowpre bool) (*Version, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   "docs.k0sproject.io",
	}

	if allowpre {
		u.Path = "latest.txt"
	} else {
		u.Path = "stable.txt"
	}

	v, err := httpGet(u.String())
	if err != nil {
		return nil, err
	}

	return NewVersion(v)
}

// LatestStable returns the semantically sorted latest non-prerelease version from the online repository
func LatestStable() (*Version, error) {
	return LatestByPrerelease(false)
}

// LatestVersion returns the semantically sorted latest version even if it is a prerelease from the online repository
func Latest() (*Version, error) {
	return LatestByPrerelease(true)
}

func httpGet(u string) (string, error) {
	client := &http.Client{
		Timeout: Timeout,
	}

	resp, err := client.Get(u)
	if err != nil {
		return "", fmt.Errorf("http request to %s failed: %w", u, err)
	}

	if resp.Body == nil {
		return "", fmt.Errorf("http request to %s failed: nil body", u)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http request to %s failed: backend returned %d", u, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http request to %s failed: %w when reading body", u, err)
	}

	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("http request to %s failed: %w when closing body", u, err)
	}

	return strings.TrimSpace(string(body)), nil
}
