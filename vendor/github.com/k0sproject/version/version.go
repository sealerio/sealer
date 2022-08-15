package version

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	goversion "github.com/hashicorp/go-version"
)

var BaseUrl = "https://github.com/k0sproject/k0s/"

// Version is a k0s version
type Version struct {
	goversion.Version
}

func pair(a, b *Version) Collection {
	return Collection{a, b}
}

// String returns a v-prefixed string representation of the k0s version
func (v *Version) String() string {
	return fmt.Sprintf("v%s", v.Version.String())
}

func (v *Version) urlString() string {
	return strings.ReplaceAll(v.String(), "+", "%2B")
}

// URL returns an URL to the release information page for the k0s version
func (v *Version) URL() string {
	return BaseUrl + filepath.Join("releases", "tag", v.urlString())
}

func (v *Version) assetBaseURL() string {
	return BaseUrl + filepath.Join("releases", "download", v.urlString()) + "/"
}

// DownloadURL returns the k0s binary download URL for the k0s version
func (v *Version) DownloadURL(os, arch string) string {
	var ext string
	if strings.HasPrefix(strings.ToLower(os), "win") {
		ext = ".exe"
	}
	return v.assetBaseURL() + fmt.Sprintf("k0s-%s-%s%s", v.String(), arch, ext)
}

// AirgapDownloadURL returns the k0s airgap bundle download URL for the k0s version
func (v *Version) AirgapDownloadURL(arch string) string {
	return v.assetBaseURL() + fmt.Sprintf("k0s-airgap-bundle-%s-%s", v.String(), arch)
}

// DocsURL returns the documentation URL for the k0s version
func (v *Version) DocsURL() string {
	return fmt.Sprintf("https://docs.k0sproject.io/%s/", v.String())
}

// Equal returns true if the version is equal to the supplied version
func (v *Version) Equal(b *Version) bool {
	return v.String() == b.String()
}

// GreaterThan returns true if the version is greater than the supplied version
func (v *Version) GreaterThan(b *Version) bool {
	if v.String() == b.String() {
		return false
	}
	p := pair(v, b)
	sort.Sort(p)
	return v.String() == p[1].String()
}

// LessThan returns true if the version is lower than the supplied version
func (v *Version) LessThan(b *Version) bool {
	if v.String() == b.String() {
		return false
	}
	return !v.GreaterThan(b)
}

// GreaterThanOrEqual returns true if the version is greater than the supplied version or equal
func (v *Version) GreaterThanOrEqual(b *Version) bool {
	return v.Equal(b) || v.GreaterThan(b)
}

// LessThanOrEqual returns true if the version is lower than the supplied version or equal
func (v *Version) LessThanOrEqual(b *Version) bool {
	return v.Equal(b) || v.LessThan(b)
}

// Compare compares two versions and returns one of the integers: -1, 0 or 1 (less than, equal, greater than)
func (v *Version) Compare(b *Version) int {
	c := v.Version.Compare(&b.Version)
	if c != 0 {
		return c
	}

	vA := v.String()

	// go to plain string comparison
	s := []string{vA, b.String()}
	sort.Strings(s)

	if vA == s[0] {
		return -1
	}

	return 1
}

// NewVersion returns a new Version created from the supplied string or an error if the string is not a valid version number
func NewVersion(v string) (*Version, error) {
	n, err := goversion.NewVersion(strings.TrimPrefix(v, "v"))
	if err != nil {
		return nil, err
	}

	return &Version{Version: *n}, nil
}
