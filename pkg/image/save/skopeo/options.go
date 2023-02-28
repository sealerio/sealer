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

package skopeo

import (
	"strconv"
	"time"

	"github.com/containers/common/pkg/retry"
)

type CopyOptions struct {
	global              *globalOptions
	deprecatedTLSVerify *deprecatedTLSVerifyOption
	srcImage            *imageOptions
	destImage           *imageDestOptions
	retryOpts           *retry.Options
	additionalTags      []string // For docker-archive: destinations, in addition to the name:tag specified as destination, also add these
	removeSignatures    bool     // Do not copy signatures from the source image
	signByFingerprint   string   // Sign the image using a GPG key with the specified fingerprint
	// new user-visible type in github.com/containers/image/v5 v5.24.0
	// https://github.com/containers/image/commit/677ec9726e33bd64076b35797cbecee75a45c368
	// signBySigstoreParamFile  string   // Sign the image using a sigstore signature per configuration in a param file
	signBySigstorePrivateKey string // Sign the image using a sigstore private key
	// signPassphraseFile string         // Path pointing to a passphrase file when signing (for either signature format, but only one of them)
	signIdentity    string         // Identity of the signed image, must be a fully specified docker reference
	digestFile      string         // Write digest to this file
	format          OptionalString // Force conversion of the image to a specified format
	all             bool           // Copy all of the images if the source is a list
	multiArch       OptionalString // How to handle multi architecture images
	preserveDigests bool           // Preserve digests during copy
	encryptLayer    []int          // The list of layers to encrypt
	encryptionKeys  []string       // Keys needed to encrypt the image
	decryptionKeys  []string       // Keys needed to decrypt the image
}

type deprecatedTLSVerifyOption struct {
	tlsVerify OptionalBool
}

// dockerImageOptions collects CLI flags specific to the "docker" transport, which are
// the same across subcommands, but may be different for each image
// (e.g. may differ between the source and destination of a copy)
type dockerImageOptions struct {
	global              *globalOptions             // May be shared across several imageOptions instances.
	shared              *sharedImageOptions        // May be shared across several imageOptions instances.
	deprecatedTLSVerify *deprecatedTLSVerifyOption // May be shared across several imageOptions instances, or nil.
	authFilePath        OptionalString             // Path to a */containers/auth.json (prefixed version to override shared image option).
	// nolint:structcheck
	credsOption OptionalString // username[:password] for accessing a registry
	// nolint:structcheck
	userName OptionalString // username for accessing a registry
	// nolint:structcheck
	password OptionalString // password for accessing a registry
	// nolint:structcheck
	registryToken OptionalString // token to be used directly as a Bearer token when accessing the registry
	// nolint:structcheck
	dockerCertPath string // A directory using Docker-like *.{crt,cert,key} files for connecting to a registry or a daemon
	// nolint:structcheck
	tlsVerify OptionalBool // Require HTTPS and verify certificates (for docker: and docker-daemon:)
	// nolint:structcheck
	noCreds bool // Access the registry anonymously
}

type globalOptions struct {
	debug              bool          // Enable debug output
	tlsVerify          OptionalBool  // Require HTTPS and verify certificates (for docker: and docker-daemon:)
	policyPath         string        // Path to a signature verification policy file
	insecurePolicy     bool          // Use an "allow everything" signature verification policy
	registriesDirPath  string        // Path to a "registries.d" registry configuration directory
	overrideArch       string        // Architecture to use for choosing images, instead of the runtime one
	overrideOS         string        // OS to use for choosing images, instead of the runtime one
	overrideVariant    string        // Architecture variant to use for choosing images, instead of the runtime one
	commandTimeout     time.Duration // Timeout for the command execution
	registriesConfPath string        // Path to the "registries.conf" file
	tmpDir             string        // Path to use for big temporary files
}

// imageOptions collects CLI flags which are the same across subcommands, but may be different for each image
// (e.g. may differ between the source and destination of a copy)
type imageOptions struct {
	dockerImageOptions
	sharedBlobDir    string // A directory to use for OCI blobs, shared across repositories
	dockerDaemonHost string // docker-daemon: host to connect to
}

// imageDestOptions is a superset of imageOptions specialized for image destinations.
// Every user should call imageDestOptions.warnAboutIneffectiveOptions() as part of handling the CLI
type imageDestOptions struct {
	*imageOptions
	dirForceCompression         bool        // Compress layers when saving to the dir: transport
	dirForceDecompression       bool        // Decompress layers when saving to the dir: transport
	ociAcceptUncompressedLayers bool        // Whether to accept uncompressed layers in the oci: transport
	compressionFormat           string      // Format to use for the compression
	compressionLevel            OptionalInt // Level to use for the compression
	precomputeDigests           bool        // Precompute digests to dedup layers when saving to the docker: transport
}

type inspectOptions struct {
	global    *globalOptions
	image     *imageOptions
	retryOpts *retry.Options
}

// sharedImageOptions collects CLI flags which are image-related, but do not change across images.
// This really should be a part of globalOptions, but that would break existing users of (skopeo copy --authfile=).
type sharedImageOptions struct {
	authFilePath string // Path to a */containers/auth.json
}

// The followings are used to replace pflag in skopeo source

// OptionalString is a string with a separate presence flag.
type OptionalString struct {
	present bool
	value   string
}

// Present returns the strings's presence flag.
func (ob *OptionalString) Present() bool {
	return ob.present
}

// Present returns the string's value. Should only be used if Present() is true.
func (ob *OptionalString) Value() string {
	return ob.value
}

func NewOptionalStringValue(p *OptionalString) {
	p.present = false
}

// Set sets the string.
func (ob *OptionalString) Set(s string) error {
	ob.value = s
	ob.present = true
	return nil
}

// OptionalBool is a boolean with a separate presence flag and value.
type OptionalBool struct {
	present bool
	value   bool
}

// Present returns the bool's presence flag.
func (ob *OptionalBool) Present() bool {
	return ob.present
}

// Value returns the bool's value. Should only be used if Present() is true.
func (ob *OptionalBool) Value() bool {
	return ob.value
}

func NewOptionalBoolValue(p *OptionalBool) {
	p.present = false
}

// Set parses the string to a bool and sets it.
func (ob *OptionalBool) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	ob.value = v
	ob.present = true
	return nil
}

// OptionalInt is a int with a separate presence flag.
type OptionalInt struct {
	present bool
	value   int
}

// Present returns the int's presence flag.
func (ob *OptionalInt) Present() bool {
	return ob.present
}

// Value returns the int's value. Should only be used if Present() is true.
func (ob *OptionalInt) Value() int {
	return ob.value
}

func NewOptionalIntValue(p *OptionalInt) {
	p.present = false
}

// Set parses the string to an int and sets it.
func (ob *OptionalInt) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	if err != nil {
		return err
	}
	ob.value = int(v)
	ob.present = true
	return nil
}
