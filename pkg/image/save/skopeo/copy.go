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
	// 1. MUST
	ctx, cancel := opts.global.commandTimeoutContext()
	defer cancel()

	// 2. MUST
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

	// signer is a new user-visible type in github.com/containers/image/v5 v5.24.0
	// used for sign destImage
	// https://github.com/containers/image/commit/677ec9726e33bd64076b35797cbecee75a45c368
	//
	//var signers []*signer.Signer
	//if opts.signBySigstoreParamFile != "" {
	//	signer, err := sigstore.NewSignerFromParameterFile(opts.signBySigstoreParamFile, &sigstore.Options{
	//		PrivateKeyPassphrasePrompt: func(keyFile string) (string, error) {
	//			return promptForPassphrase(keyFile, os.Stdin, os.Stdout)
	//		},
	//		Stdin:  os.Stdin,
	//		Stdout: os.Stdout,
	//	})
	//	if err != nil {
	//		return fmt.Errorf("Error using --sign-by-sigstore: %w", err)
	//	}
	//	defer signer.Close()
	//	signers = append(signers, signer)
	//}

	// c/image/copy.Image does allow creating both simple signing and sigstore signatures simultaneously,
	// with independent passphrases, but that would make the CLI probably too confusing.
	// For now, use the passphrase with either, but only one of them.
	//if opts.signPassphraseFile != "" && opts.signByFingerprint != "" && opts.signBySigstorePrivateKey != "" {
	//	return fmt.Errorf("Only one of --sign-by and sign-by-sigstore-private-key can be used with sign-passphrase-file")
	//}

	// passphrase used fof sign
	var passphrase string
	//if opts.signPassphraseFile != "" {
	//	p, err := cli.ReadPassphraseFile(opts.signPassphraseFile)
	//	if err != nil {
	//		return err
	//	}
	//	passphrase = p
	//} else if opts.signBySigstorePrivateKey != "" {
	//	p, err := promptForPassphrase(opts.signBySigstorePrivateKey, os.Stdin, os.Stdout)
	//	if err != nil {
	//		return err
	//	}
	//	passphrase = p
	//} // opts.signByFingerprint triggers a GPG-agent passphrase prompt, possibly using a more secure channel, so we usually shouldn’t prompt ourselves if no passphrase was explicitly provided.

	// signIdentity used for sign
	var signIdentity reference.Named = nil
	//if opts.signIdentity != "" {
	//	signIdentity, err = reference.ParseNamed(opts.signIdentity)
	//	if err != nil {
	//		return fmt.Errorf("Could not parse --sign-identity: %v", err)
	//	}
	//}

	// 7. MUST，好像这里可以设置架构，需要测试
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

	// 默认是空值，这里应该是暂时用不上的
	var manifestType string
	if opts.format.Present() {
		manifestType, err = parseManifestFormat(opts.format.Value())
		if err != nil {
			return err
		}
	}
	// 还是这里可以设置架构？需要测试
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
			RemoveSignatures: opts.removeSignatures,
			//Signers:                          signers,
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
		global:              global,
		deprecatedTLSVerify: deprecatedTLSVerifyOpt,
		srcImage:            srcOpts,
		destImage:           destOpts,
		retryOpts:           &retry.Options{},
		additionalTags:      []string{},
		removeSignatures:    false,
		signByFingerprint:   "",
		//signBySigstoreParamFile:  "",
		signBySigstorePrivateKey: "",
		//signPassphraseFile:       "",
		signIdentity:    "",
		digestFile:      "",
		all:             false,
		preserveDigests: false,
		encryptLayer:    []int{},
		encryptionKeys:  []string{},
		decryptionKeys:  []string{},
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
