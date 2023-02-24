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

package buildah

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah/util"
	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) LookupManifest(name string) (*libimage.ManifestList, error) {
	return engine.libimageRuntime.LookupManifestList(name)
}

func (engine *Engine) CreateManifest(name string, opts *options.ManifestCreateOpts) error {
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()
	list := manifests.Create()

	names, err := util.ExpandNames([]string{name}, systemCxt, store)
	if err != nil {
		return fmt.Errorf("encountered while expanding image name %q: %w", name, err)
	}
	_, err = list.SaveToImage(store, "", names, manifest.DockerV2ListMediaType)

	return err
}

func (engine *Engine) DeleteManifests(names []string, opts *options.ManifestDeleteOpts) error {
	runtime := engine.ImageRuntime()

	rmiReports, rmiErrors := runtime.RemoveImages(context.Background(), names, &libimage.RemoveImagesOptions{
		Filters:        []string{"readonly=false"},
		LookupManifest: true,
	})
	for _, r := range rmiReports {
		for _, u := range r.Untagged {
			logrus.Infof("untagged: %s", u)
		}
	}
	for _, r := range rmiReports {
		if r.Removed {
			logrus.Infof("%s", r.ID)
		}
	}

	var multiE *multierror.Error
	multiE = multierror.Append(multiE, rmiErrors...)
	return multiE.ErrorOrNil()
}

func (engine *Engine) InspectManifest(name string, opts *options.ManifestInspectOpts) error {
	printManifest := func(manifest []byte) error {
		var b bytes.Buffer
		err := json.Indent(&b, manifest, "", "    ")
		if err != nil {
			return fmt.Errorf("rendering manifest for display: %w", err)
		}

		logrus.Infof("%s", b.String())
		return nil
	}

	runtime := engine.ImageRuntime()
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()
	ctx := getContext()

	// Before doing a remote lookup, attempt to resolve the manifest list
	// locally.
	manifestList, err := runtime.LookupManifestList(name)
	if err == nil {
		schema2List, err := manifestList.Inspect()
		if err != nil {
			return err
		}

		rawSchema2List, err := json.Marshal(schema2List)
		if err != nil {
			return err
		}

		return printManifest(rawSchema2List)
	}
	if !errors.Is(err, storage.ErrImageUnknown) && !errors.Is(err, libimage.ErrNotAManifestList) {
		return err
	}

	// TODO: at some point `libimage` should support resolving manifests
	// like that.  Similar to `libimage.Runtime.LookupImage` we could
	// implement a `*.LookupImageIndex`.
	refs, err := util.ResolveNameToReferences(store, systemCxt, name)
	if err != nil {
		logrus.Debugf("error parsing reference to image %q: %v", name, err)
	}

	if ref, _, err := util.FindImage(store, "", systemCxt, name); err == nil {
		refs = append(refs, ref)
	} else if ref, err := alltransports.ParseImageName(name); err == nil {
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return fmt.Errorf("locating images with names %v", name)
	}

	var (
		latestErr error
		result    []byte
	)

	appendErr := func(e error) {
		if latestErr == nil {
			latestErr = e
		} else {
			latestErr = fmt.Errorf("tried %v: %w", e, latestErr)
		}
	}

	for _, ref := range refs {
		logrus.Debugf("Testing reference %q for possible manifest", transports.ImageName(ref))

		src, err := ref.NewImageSource(ctx, systemCxt)
		if err != nil {
			appendErr(fmt.Errorf("reading image %q: %w", transports.ImageName(ref), err))
			continue
		}
		defer src.Close()

		manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
		if err != nil {
			appendErr(fmt.Errorf("loading manifest %q: %w", transports.ImageName(ref), err))
			continue
		}

		if !manifest.MIMETypeIsMultiImage(manifestType) {
			appendErr(fmt.Errorf("manifest is of type %s (not a list type)", manifestType))
			continue
		}
		result = manifestBytes
		break
	}
	if len(result) == 0 && latestErr != nil {
		return latestErr
	}

	return printManifest(result)
}

func (engine *Engine) PushManifest(name, destSpec string, opts *options.PushOptions) error {
	runtime := engine.ImageRuntime()
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()

	manifestList, err := runtime.LookupManifestList(name)
	if err != nil {
		return err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	dest, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		destTransport := strings.Split(destSpec, ":")[0]
		if t := transports.Get(destTransport); t != nil {
			return err
		}

		if strings.Contains(destSpec, "://") {
			return err
		}

		destSpec = "docker://" + destSpec
		dest2, err2 := alltransports.ParseImageName(destSpec)
		if err2 != nil {
			return err
		}
		dest = dest2
		logrus.Debugf("Assuming docker:// as the transport method for DESTINATION: %s", destSpec)
	}

	var manifestType string
	if opts.Format != "" {
		switch opts.Format {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return fmt.Errorf("unknown format %q. Choose on of the supported formats: 'oci' or 'v2s2'", opts.Format)
		}
	}
	pushOptions := manifests.PushOptions{
		Store:              store,
		SystemContext:      systemCxt,
		ImageListSelection: cp.CopySystemImage,
		Instances:          nil,
		ManifestType:       manifestType,
	}
	if opts.All {
		pushOptions.ImageListSelection = cp.CopyAllImages
	}
	if !opts.Quiet {
		pushOptions.ReportWriter = os.Stderr
	}

	_, _, err = list.Push(getContext(), dest, pushOptions)

	if err == nil && opts.Rm {
		_, err = store.DeleteImage(manifestList.ID(), true)
	}

	return err
}

func (engine *Engine) AddToManifest(name, imageSpec string, opts *options.ManifestAddOpts) error {
	runtime := engine.ImageRuntime()
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()

	manifestList, err := runtime.LookupManifestList(name)
	if err != nil {
		return err
	}
	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	ref, err := alltransports.ParseImageName(imageSpec)
	if err != nil {
		if ref, err = alltransports.ParseImageName(util.DefaultTransport + imageSpec); err != nil {
			// check if the local image exists
			if ref, _, err = util.FindImage(store, "", systemCxt, imageSpec); err != nil {
				return err
			}
		}
	}

	digestID, err := list.Add(getContext(), systemCxt, ref, opts.All)
	if err != nil {
		var storeErr error
		// Retry without a custom system context.  A user may want to add
		// a custom platform (see #3511).
		if ref, _, storeErr = util.FindImage(store, "", nil, imageSpec); storeErr != nil {
			logrus.Errorf("Error while trying to find image on local storage: %v", storeErr)
			return err
		}
		digestID, storeErr = list.Add(getContext(), systemCxt, ref, opts.All)
		if storeErr != nil {
			logrus.Errorf("Error while trying to add on manifest list: %v", storeErr)
			return err
		}
	}

	if opts.Os != "" {
		if err = list.SetOS(digestID, opts.Os); err != nil {
			return err
		}
	}
	if opts.OsVersion != "" {
		if err = list.SetOSVersion(digestID, opts.OsVersion); err != nil {
			return err
		}
	}
	if len(opts.OsFeatures) != 0 {
		if err = list.SetOSFeatures(digestID, opts.OsFeatures); err != nil {
			return err
		}
	}
	if opts.Arch != "" {
		if err = list.SetArchitecture(digestID, opts.Arch); err != nil {
			return err
		}
	}
	if opts.Variant != "" {
		if err = list.SetVariant(digestID, opts.Variant); err != nil {
			return err
		}
	}

	if len(opts.Annotations) != 0 {
		annotations := make(map[string]string)
		for _, annotationSpec := range opts.Annotations {
			spec := strings.SplitN(annotationSpec, "=", 2)
			if len(spec) != 2 {
				return fmt.Errorf("no value given for annotation %q", spec[0])
			}
			annotations[spec[0]] = spec[1]
		}
		if err = list.SetAnnotations(&digestID, annotations); err != nil {
			return err
		}
	}

	_, err = list.SaveToImage(store, manifestList.ID(), nil, "")

	return err
}

func (engine *Engine) RemoveFromManifest(name string, instanceDigest digest.Digest, opts *options.ManifestRemoveOpts) error {
	runtime := engine.ImageRuntime()

	manifestList, err := runtime.LookupManifestList(name)
	if err != nil {
		return err
	}

	if err = manifestList.RemoveInstance(instanceDigest); err != nil {
		return err
	}

	logrus.Infof("%s: %s", manifestList.ID(), instanceDigest.String())

	return nil
}
