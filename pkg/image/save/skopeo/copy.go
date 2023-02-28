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
	"fmt"
	"os"

	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	encconfig "github.com/containers/ocicrypt/config"
	enchelpers "github.com/containers/ocicrypt/helpers"

	utilsos "github.com/sealerio/sealer/utils/os"
)

func Copy(srcImage, destImage string, creds string) (retErr error) {
	opts, err := NewCopyOptions(creds)
	if err != nil {
		return err
	}

	ctx, cancel := opts.global.commandTimeoutContext()
	defer cancel()

	policyContext, err := opts.global.getPolicyContext()
	if err != nil {
		return fmt.Errorf("error loading trust policy: %v", err)
	}
	defer func() {
		if err := policyContext.Destroy(); err != nil {
			retErr = noteCloseFailure(retErr, "tearing down policy context", err)
		}
	}()

	srcRef, err := alltransports.ParseImageName(srcImage)
	if err != nil {
		return fmt.Errorf("invalid source name %s: %v", srcImage, err)
	}
	destRef, err := alltransports.ParseImageName(destImage)
	if err != nil {
		return fmt.Errorf("invalid destination name %s: %v", destImage, err)
	}

	// passphrase used for sign, it is not needed to set value now.
	var passphrase string

	// signIdentity used for sign, it is not needed to set value now.
	var signIdentity reference.Named = nil

	// TODO: It looks like that can set up the architecture here and need to test it
	sourceCtx, err := opts.srcImage.newSystemContext()
	if err != nil {
		return err
	}
	destinationCtx, err := opts.destImage.newSystemContext()
	if err != nil {
		return err
	}
	for _, image := range opts.additionalTags {
		ref, err := reference.ParseNormalizedNamed(image)
		if err != nil {
			return fmt.Errorf("error parsing additional-tag '%s': %v", image, err)
		}
		namedTagged, isNamedTagged := ref.(reference.NamedTagged)
		if !isNamedTagged {
			return fmt.Errorf("additional-tag '%s' must be a tagged reference", image)
		}
		destinationCtx.DockerArchiveAdditionalTags = append(destinationCtx.DockerArchiveAdditionalTags, namedTagged)
	}

	//  manifest MIME type of image set by user. "" is default and means use the autodetection to the the manifest MIME type
	//  It is not needed to set value now.
	var manifestType string
	if opts.format.Present() {
		manifestType, err = parseManifestFormat(opts.format.Value())
		if err != nil {
			return err
		}
	}

	// TODO: It looks like that can set up the architecture here and need to test it
	imageListSelection := copy.CopySystemImage
	if opts.multiArch.Present() && opts.all {
		return fmt.Errorf("cannot use --all and --multi-arch flags together")
	}
	if opts.multiArch.Present() {
		imageListSelection, err = parseMultiArch(opts.multiArch.Value())
		if err != nil {
			return err
		}
	}
	if opts.all {
		imageListSelection = copy.CopyAllImages
	}

	if len(opts.encryptionKeys) > 0 && len(opts.decryptionKeys) > 0 {
		return fmt.Errorf("--encryption-key and --decryption-key cannot be specified together")
	}
	var encLayers *[]int
	var encConfig *encconfig.EncryptConfig
	var decConfig *encconfig.DecryptConfig
	if len(opts.encryptLayer) > 0 && len(opts.encryptionKeys) == 0 {
		return fmt.Errorf("--encrypt-layer can only be used with --encryption-key")
	}
	if len(opts.encryptionKeys) > 0 {
		// encryption
		p := opts.encryptLayer
		encLayers = &p
		encryptionKeys := opts.encryptionKeys
		ecc, err := enchelpers.CreateCryptoConfig(encryptionKeys, []string{})
		if err != nil {
			return fmt.Errorf("invalid encryption keys: %v", err)
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{ecc})
		encConfig = cc.EncryptConfig
	}
	if len(opts.decryptionKeys) > 0 {
		// decryption
		decryptionKeys := opts.decryptionKeys
		dcc, err := enchelpers.CreateCryptoConfig([]string{}, decryptionKeys)
		if err != nil {
			return fmt.Errorf("invalid decryption keys: %v", err)
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{dcc})
		decConfig = cc.DecryptConfig
	}

	return retry.IfNecessary(ctx, func() error {
		manifestBytes, err := copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
			RemoveSignatures:                 opts.removeSignatures,
			SignBy:                           opts.signByFingerprint,
			SignPassphrase:                   passphrase,
			SignBySigstorePrivateKeyFile:     opts.signBySigstorePrivateKey,
			SignSigstorePrivateKeyPassphrase: []byte(passphrase),
			SignIdentity:                     signIdentity,
			ReportWriter:                     os.Stdout,
			SourceCtx:                        sourceCtx,
			DestinationCtx:                   destinationCtx,
			ForceManifestMIMEType:            manifestType,
			ImageListSelection:               imageListSelection,
			PreserveDigests:                  opts.preserveDigests,
			OciDecryptConfig:                 decConfig,
			OciEncryptLayers:                 encLayers,
			OciEncryptConfig:                 encConfig,
		})
		if err != nil {
			return err
		}
		if opts.digestFile != "" {
			manifestDigest, err := manifest.Digest(manifestBytes)
			if err != nil {
				return err
			}
			if err = utilsos.NewCommonWriter(opts.digestFile).WriteFile([]byte(manifestDigest.String())); err != nil {
				return fmt.Errorf("failed to write digest to file %q: %w", opts.digestFile, err)
			}
		}
		return nil
	}, opts.retryOpts)
}

func NewCopyOptions(creds string) (CopyOptions, error) {
	global := initGlobal()
	// Path to a */containers/auth.json
	sharedOpts := &sharedImageOptions{os.Getenv("REGISTRY_AUTH_FILE")}
	deprecatedTLSVerifyOpt := &deprecatedTLSVerifyOption{}
	srcOpts := imageFlags(global, sharedOpts, deprecatedTLSVerifyOpt)
	destOpts := imageDestFlags(global, sharedOpts, deprecatedTLSVerifyOpt)

	// belows are default value in skopeo
	opts := CopyOptions{
		global:                   global,
		deprecatedTLSVerify:      deprecatedTLSVerifyOpt,
		srcImage:                 srcOpts,
		destImage:                destOpts,
		retryOpts:                &retry.Options{},
		additionalTags:           []string{},
		removeSignatures:         false,
		signByFingerprint:        "",
		signBySigstorePrivateKey: "",
		signIdentity:             "",
		digestFile:               "",
		all:                      false,
		preserveDigests:          false,
		encryptLayer:             []int{},
		encryptionKeys:           []string{},
		decryptionKeys:           []string{},
	}
	NewOptionalStringValue(&opts.multiArch)
	NewOptionalStringValue(&opts.format)
	if creds != "" {
		if err := opts.srcImage.credsOption.Set(creds); err != nil {
			return opts, err
		}
	}
	return opts, nil
}
