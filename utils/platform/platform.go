// Copyright © 2022 Alibaba Group Holding Ltd.
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

package platform

import (
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

var (
	specifierRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func ParsePlatforms(v string) ([]*v1.Platform, error) {
	var pp []*v1.Platform
	for _, v := range strings.Split(v, ",") {
		p, err := Parse(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse target platform %s", v)
		}
		p = Normalize(p)
		pp = append(pp, &p)
	}

	return pp, nil
}

func Normalize(platform v1.Platform) v1.Platform {
	platform.OS = normalizeOS(platform.OS)
	platform.Architecture, platform.Variant = NormalizeArch(platform.Architecture, platform.Variant)

	return platform
}

func Parse(specifier string) (v1.Platform, error) {
	if strings.Contains(specifier, "*") {
		return v1.Platform{}, errors.Wrapf(ErrInvalidArgument, "%q: wildcards not yet supported", specifier)
	}

	parts := strings.Split(specifier, "/")

	for _, part := range parts {
		if !specifierRe.MatchString(part) {
			return v1.Platform{}, errors.Wrapf(ErrInvalidArgument, "%q is an invalid component of %q: platform specifier component must match %q", part, specifier, specifierRe.String())
		}
	}

	var p v1.Platform
	switch len(parts) {
	case 1:
		// in this case, we will test that the value might be an OS, then look
		// it up. If it is not known, we'll treat it as an architecture. Since
		// we have very little information about the platform here, we are
		// going to be a little stricter if we don't know about the argument
		// value.
		p.OS = normalizeOS(parts[0])
		if isKnownOS(p.OS) {
			// picks a default architecture
			p.Architecture = runtime.GOARCH
			if p.Architecture == ARM && cpuVariant() != "v7" {
				p.Variant = cpuVariant()
			}

			return p, nil
		}

		p.Architecture, p.Variant = NormalizeArch(parts[0], "")
		if p.Architecture == ARM && p.Variant == "v7" {
			p.Variant = ""
		}
		if isKnownArch(p.Architecture) {
			p.OS = runtime.GOOS
			return p, nil
		}

		return v1.Platform{}, errors.Wrapf(ErrInvalidArgument, "%q: unknown operating system or architecture", specifier)
	case 2:
		// In this case, we treat as a regular os/arch pair. We don't care
		// about whether we know of the platform.
		p.OS = normalizeOS(parts[0])
		p.Architecture, p.Variant = NormalizeArch(parts[1], "")
		if p.Architecture == ARM && p.Variant == "v7" {
			p.Variant = ""
		}

		return p, nil
	case 3:
		// we have a fully specified variant, this is rare
		p.OS = normalizeOS(parts[0])
		p.Architecture, p.Variant = NormalizeArch(parts[1], parts[2])
		if p.Architecture == ARM64 && p.Variant == "" {
			p.Variant = "v8"
		}

		return p, nil
	}

	return v1.Platform{}, errors.Wrapf(ErrInvalidArgument, "%q: cannot parse platform specifier", specifier)
}

func GetDefaultPlatform() v1.Platform {
	return v1.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		// The Variant field will be empty if arch != ARM.
		Variant: cpuVariant(),
	}
}

// Format returns a string specifier from the provided platform specification.
func Format(platform v1.Platform) string {
	if platform.OS == "" {
		return "unknown"
	}

	return path.Join(platform.OS, platform.Architecture, platform.Variant)
}

// Matched check if src == dest
func Matched(src, dest v1.Platform) bool {
	if src.OS == dest.OS &&
		src.Architecture == ARM64 && dest.Architecture == ARM64 {
		return true
	}

	return src.OS == dest.OS &&
		src.Architecture == dest.Architecture &&
		src.Variant == dest.Variant
}

func isLinuxOS(os string) bool {
	return os == "linux"
}

// NormalizeArch normalizes the architecture.
func NormalizeArch(arch, variant string) (string, string) {
	arch, variant = strings.ToLower(arch), strings.ToLower(variant)
	switch arch {
	case "i386":
		arch = "386"
		variant = ""
	case "x86_64", "x86-64":
		arch = "amd64"
		variant = ""
	//nolint
	case "aarch64", "arm64":
		arch = "arm64"
		variant = "v8"
	case "armhf":
		//nolint
		arch = "arm"
		variant = "v7"
	case "armel":
		arch = "arm"
		variant = "v6"
	case "arm":
		switch variant {
		case "", "7":
			variant = "v7"
		case "5", "6", "8":
			variant = "v" + variant
		}
	}

	return arch, variant
}

func isKnownOS(os string) bool {
	switch os {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "windows", "zos":
		return true
	}
	return false
}

func isArmArch(arch string) bool {
	switch arch {
	case "arm", "arm64":
		return true
	}
	return false
}

func isKnownArch(arch string) bool {
	switch arch {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be",
		"ppc64", "ppc64le", "loong64", "mips", "mipsle", "mips64", "mips64le", "mips64p32",
		"mips64p32le", "ppc", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	}
	return false
}

func normalizeOS(os string) string {
	if os == "" {
		return runtime.GOOS
	}
	os = strings.ToLower(os)

	switch os {
	case "macos":
		os = "darwin"
	}
	return os
}
