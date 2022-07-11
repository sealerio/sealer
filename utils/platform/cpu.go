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
	"bufio"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	ARM     = "arm"
	ARM64   = "arm64"
	AMD     = "amd64"
	UNKNOWN = "unknown"
	WINDOWS = "windows"
	DARWIN  = "darwin"
	LINUX   = "linux"
)

var (
	cpuVariantValue string
	cpuVariantOnce  sync.Once
)

func cpuVariant() string {
	cpuVariantOnce.Do(func() {
		if isArmArch(runtime.GOARCH) {
			variant, err := getCPUInfo("Cpu architecture")
			if err != nil {
				logrus.Error(err)
			}
			model, err := getCPUInfo("model name")
			if err != nil && !strings.Contains(err.Error(), ErrNotFound.Error()) {
				logrus.Error(err)
			}
			cpuVariantValue = GetCPUVariantByInfo(runtime.GOOS, runtime.GOARCH, variant, model)
		}
	})
	return cpuVariantValue
}

// For Linux, We can just parse this information from "/proc/cpuinfo"
func getCPUInfo(pattern string) (info string, err error) {
	if !isLinuxOS(runtime.GOOS) {
		return "", errors.Wrapf(ErrNotImplemented, "getCPUInfo for OS %s", runtime.GOOS)
	}

	cpuinfo, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := cpuinfo.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(cpuinfo)
	for scanner.Scan() {
		newline := scanner.Text()
		list := strings.Split(newline, ":")

		if len(list) > 1 && strings.EqualFold(strings.TrimSpace(list[0]), pattern) {
			return strings.TrimSpace(list[1]), nil
		}
	}

	// Check whether the scanner encountered errors
	err = scanner.Err()
	if err != nil {
		return "", err
	}

	return "", errors.Wrapf(ErrNotFound, "getCPUInfo for pattern: %s", pattern)
}

// GetCPUVariantByInfo get 'Cpu architecture', 'model name' from /proc/cpuinfo
func GetCPUVariantByInfo(os, arch, variant, model string) string {
	if os == WINDOWS || os == DARWIN {
		// Windows/Darwin only supports v7 for ARM32 and v8 for ARM64, and so we can use
		// runtime.GOARCH to determine the variants
		var variant string
		switch arch {
		case ARM64:
			variant = "v8"
		case ARM:
			variant = "v7"
		default:
			variant = UNKNOWN
		}

		return variant
	}

	if arch == ARM && variant == "7" {
		if strings.HasPrefix(strings.ToLower(model), "armv6-compatible") {
			variant = "6"
		}
	}

	switch strings.ToLower(variant) {
	case "8", "aarch64":
		variant = "v8"
	case "7", "7m", "?(12)", "?(13)", "?(14)", "?(15)", "?(16)", "?(17)":
		variant = "v7"
	case "6", "6tej":
		variant = "v6"
	case "5", "5t", "5te", "5tej":
		variant = "v5"
	case "4", "4t":
		variant = "v4"
	case "3":
		variant = "v3"
	default:
		variant = "unknown"
	}

	return variant
}
