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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func initGlobal() *globalOptions {
	opt := &globalOptions{
		debug:              false,
		policyPath:         "",
		insecurePolicy:     false,
		registriesDirPath:  "",
		registriesConfPath: "",
		overrideArch:       "",
		overrideOS:         "",
		overrideVariant:    "",
		commandTimeout:     0,
		tmpDir:             "",
	}
	NewOptionalBoolValue(&opt.tlsVerify)
	return opt
}

// commandTimeoutContext returns a context.Context and a cancellation callback based on opts.
// The caller should usually "defer cancel()" immediately after calling this.
func (opts *globalOptions) commandTimeoutContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	var cancel context.CancelFunc = func() {}
	if opts.commandTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.commandTimeout)
	}
	return ctx, cancel
}

// newSystemContext returns a *types.SystemContext corresponding to opts.
// It is guaranteed to return a fresh instance, so it is safe to make additional updates to it.
func (opts *globalOptions) newSystemContext() *types.SystemContext {
	ctx := &types.SystemContext{
		RegistriesDirPath:        opts.registriesDirPath,
		ArchitectureChoice:       opts.overrideArch,
		OSChoice:                 opts.overrideOS,
		VariantChoice:            opts.overrideVariant,
		SystemRegistriesConfPath: opts.registriesConfPath,
		BigFilesTemporaryDir:     opts.tmpDir,
		DockerRegistryUserAgent:  defaultUserAgent,
	}
	// DEPRECATED: We support this for backward compatibility, but override it if a per-image flag is provided.
	if opts.tlsVerify.Present() {
		ctx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!opts.tlsVerify.Value())
	}
	return ctx
}

// getPolicyContext returns a *signature.PolicyContext based on opts.
func (opts *globalOptions) getPolicyContext() (*signature.PolicyContext, error) {
	var policy *signature.Policy // This could be cached across calls in opts.
	var err error
	if opts.insecurePolicy {
		policy = &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	} else if opts.policyPath == "" {
		policy, err = signature.DefaultPolicy(nil)
	} else {
		policy, err = signature.NewPolicyFromFile(opts.policyPath)
	}
	if err != nil {
		return nil, err
	}
	return signature.NewPolicyContext(policy)
}

// newSystemContext returns a *types.SystemContext corresponding to opts.
// It is guaranteed to return a fresh instance, so it is safe to make additional updates to it.
func (opts *imageOptions) newSystemContext() (*types.SystemContext, error) {
	// *types.SystemContext instance from globalOptions
	//  imageOptions option overrides the instance if both are present.
	ctx := opts.global.newSystemContext()
	ctx.DockerCertPath = opts.dockerCertPath
	ctx.OCISharedBlobDirPath = opts.sharedBlobDir
	ctx.AuthFilePath = opts.shared.authFilePath
	ctx.DockerDaemonHost = opts.dockerDaemonHost
	ctx.DockerDaemonCertPath = opts.dockerCertPath
	ctx.ArchitectureChoice = ""
	if opts.dockerImageOptions.authFilePath.Present() {
		ctx.AuthFilePath = opts.dockerImageOptions.authFilePath.Value()
	}
	if opts.deprecatedTLSVerify != nil && opts.deprecatedTLSVerify.tlsVerify.Present() {
		// If both this deprecated option and a non-deprecated option is present, we use the latter value.
		ctx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!opts.deprecatedTLSVerify.tlsVerify.Value())
	}
	if opts.tlsVerify.Present() {
		ctx.DockerDaemonInsecureSkipTLSVerify = !opts.tlsVerify.Value()
	}
	if opts.tlsVerify.Present() {
		ctx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!opts.tlsVerify.Value())
	}
	if opts.credsOption.Present() && opts.noCreds {
		return nil, errors.New("creds and no-creds cannot be specified at the same time")
	}
	if opts.userName.Present() && opts.noCreds {
		return nil, errors.New("username and no-creds cannot be specified at the same time")
	}
	if opts.credsOption.Present() && opts.userName.Present() {
		return nil, errors.New("creds and username cannot be specified at the same time")
	}
	// if any of username or password is present, then both are expected to be present
	if opts.userName.Present() != opts.password.Present() {
		if opts.userName.Present() {
			return nil, errors.New("password must be specified when username is specified")
		}
		return nil, errors.New("username must be specified when password is specified")
	}
	if opts.credsOption.Present() {
		var err error
		ctx.DockerAuthConfig, err = getDockerAuth(opts.credsOption.Value())
		if err != nil {
			return nil, err
		}
	} else if opts.userName.Present() {
		ctx.DockerAuthConfig = &types.DockerAuthConfig{
			Username: opts.userName.Value(),
			Password: opts.password.Value(),
		}
	}
	if opts.registryToken.Present() {
		ctx.DockerBearerRegistryToken = opts.registryToken.Value()
	}
	if opts.noCreds {
		ctx.DockerAuthConfig = &types.DockerAuthConfig{}
	}

	return ctx, nil
}

// imageFlags prepares a collection of CLI flags writing into imageOptions, and the managed imageOptions structure.
//func imageFlags(global *globalOptions, shared *sharedImageOptions, deprecatedTLSVerify *deprecatedTLSVerifyOption, flagPrefix, credsOptionAlias string) (pflag.FlagSet, *imageOptions) {
func imageFlags(global *globalOptions, shared *sharedImageOptions, deprecatedTLSVerify *deprecatedTLSVerifyOption) *imageOptions {
	opts := dockerImageFlags(global, shared, deprecatedTLSVerify)
	opts.sharedBlobDir = ""
	opts.dockerDaemonHost = ""
	return opts
}

// imageDestFlags prepares a collection of CLI flags writing into imageDestOptions, and the managed imageDestOptions structure.
func imageDestFlags(global *globalOptions, shared *sharedImageOptions, deprecatedTLSVerify *deprecatedTLSVerifyOption) *imageDestOptions {
	genericOptions := imageFlags(global, shared, deprecatedTLSVerify)
	opts := imageDestOptions{imageOptions: genericOptions}
	opts.dirForceCompression = false
	opts.dirForceDecompression = false
	opts.ociAcceptUncompressedLayers = false
	opts.compressionFormat = ""
	NewOptionalIntValue(&opts.compressionLevel)
	opts.precomputeDigests = false
	return &opts
}

func IsImageExist(imageName string) bool {
	global := initGlobal()
	sharedOpts := &sharedImageOptions{os.Getenv("REGISTRY_AUTH_FILE")}
	imageOpts := imageFlags(global, sharedOpts, nil)
	retryOpts := &retry.Options{MaxRetry: 0}
	opts := &inspectOptions{
		global:    global,
		image:     imageOpts,
		retryOpts: retryOpts,
	}

	ctx, cancel := opts.global.commandTimeoutContext()
	defer cancel()

	var (
		retErr error
		src    types.ImageSource
	)
	if err := retry.IfNecessary(ctx, func() error {
		src, retErr = parseImageSource(ctx, opts.image, imageName)
		return retErr
	}, opts.retryOpts); err != nil {
		return false
	}
	defer func() {
		if err := src.Close(); err != nil {
			retErr = noteCloseFailure(retErr, "closing image", err)
		}
	}()
	return true
}

// dockerImageFlags prepares a collection of docker-transport specific CLI flags
// writing into imageOptions, and the managed imageOptions structure.
func dockerImageFlags(global *globalOptions, shared *sharedImageOptions, deprecatedTLSVerify *deprecatedTLSVerifyOption) *imageOptions {
	flags := imageOptions{
		dockerImageOptions: dockerImageOptions{
			global:              global,
			shared:              shared,
			deprecatedTLSVerify: deprecatedTLSVerify,
		},
	}
	NewOptionalStringValue(&flags.authFilePath)
	NewOptionalStringValue(&flags.credsOption)
	NewOptionalStringValue(&flags.userName)
	NewOptionalStringValue(&flags.password)
	NewOptionalStringValue(&flags.registryToken)
	NewOptionalBoolValue(&flags.tlsVerify)
	flags.dockerCertPath = ""
	flags.noCreds = false

	return &flags
}
func getDockerAuth(creds string) (*types.DockerAuthConfig, error) {
	username, password, err := parseCreds(creds)
	if err != nil {
		return nil, err
	}
	return &types.DockerAuthConfig{
		Username: username,
		Password: password,
	}, nil
}

// noteCloseFailure returns (possibly-nil) err modified to account for (non-nil) closeErr.
// The error for closeErr is annotated with description (which is not a format string)
// Typical usage:
//
//	defer func() {
//		if err := something.Close(); err != nil {
//			returnedErr = noteCloseFailure(returnedErr, "closing something", err)
//		}
//	}
func noteCloseFailure(err error, description string, closeErr error) error {
	// We don’t accept a Closer() and close it ourselves because signature.PolicyContext has .Destroy(), not .Close().
	// This also makes it harder for a caller to do
	//     defer noteCloseFailure(returnedErr, …)
	// which doesn’t use the right value of returnedErr, and doesn’t update it.
	if err == nil {
		return fmt.Errorf("%s: %w", description, closeErr)
	}
	// In this case we prioritize the primary error for use with %w; closeErr is usually less relevant, or might be a consequence of the primary erorr.
	return fmt.Errorf("%w (%s: %v)", err, description, closeErr)
}

// Cut is Go 1.18 new strings feature, here for compatibility
func Cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func parseCreds(creds string) (string, string, error) {
	if creds == "" {
		return "", "", errors.New("credentials can't be empty")
	}
	username, password, _ := Cut(creds, ":") // Sets password to "" if there is no ":"
	if username == "" {
		return "", "", errors.New("username can't be empty")
	}
	return username, password, nil
}

// parseImageSource converts image URL-like string to an ImageSource.
// The caller must call .Close() on the returned ImageSource.
func parseImageSource(ctx context.Context, opts *imageOptions, name string) (types.ImageSource, error) {
	ref, err := alltransports.ParseImageName(name)
	if err != nil {
		return nil, err
	}
	sys, err := opts.newSystemContext()
	if err != nil {
		return nil, err
	}
	return ref.NewImageSource(ctx, sys)
}

// 暂时应该是用不上的
// parseManifestFormat parses format parameter for copy and sync command.
// It returns string value to use as manifest MIME type
func parseManifestFormat(manifestFormat string) (string, error) {
	switch manifestFormat {
	case "oci":
		return imgspecv1.MediaTypeImageManifest, nil
	case "v2s1":
		return manifest.DockerV2Schema1SignedMediaType, nil
	case "v2s2":
		return manifest.DockerV2Schema2MediaType, nil
	default:
		return "", fmt.Errorf("unknown format %q. Choose one of the supported formats: 'oci', 'v2s1', or 'v2s2'", manifestFormat)
	}
}

// parseMultiArch parses the list processing selection
// It returns the copy.ImageListSelection to use with image.Copy option
func parseMultiArch(multiArch string) (copy.ImageListSelection, error) {
	switch multiArch {
	case "system":
		return copy.CopySystemImage, nil
	case "all":
		return copy.CopyAllImages, nil
	// There is no CopyNoImages value in copy.ImageListSelection, but because we
	// don't provide an option to select a set of images to copy, we can use
	// CopySpecificImages.
	case "index-only":
		return copy.CopySpecificImages, nil
	// We don't expose CopySpecificImages other than index-only above, because
	// we currently don't provide an option to choose the images to copy. That
	// could be added in the future.
	default:
		return copy.CopySystemImage, fmt.Errorf("unknown multi-arch option %q. Choose one of the supported options: 'system', 'all', or 'index-only'", multiArch)
	}
}
