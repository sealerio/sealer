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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containers/storage"
	"github.com/pkg/errors"

	"github.com/containers/buildah/util"
	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sirupsen/logrus"
)

func (engine *Engine) LookupManifest(name string) (*libimage.ManifestList, error) {
	return engine.libimageRuntime.LookupManifestList(name)
}

func (engine *Engine) CreateManifest(name string, opts *options.ManifestCreateOpts) (string, error) {
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()
	list := manifests.Create()

	names, err := util.ExpandNames([]string{name}, systemCxt, store)
	if err != nil {
		return "", fmt.Errorf("encountered while expanding image name %q: %w", name, err)
	}

	return list.SaveToImage(store, "", names, manifest.DockerV2ListMediaType)
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

func (engine *Engine) InspectManifest(name string, opts *options.ManifestInspectOpts) (*libimage.ManifestListData, error) {
	runtime := engine.ImageRuntime()

	// attempt to resolve the manifest list locally.
	manifestList, err := runtime.LookupManifestList(name)
	if err != nil {
		return nil, err
	}

	return manifestList.Inspect()
}

func (engine *Engine) PushManifest(name, destSpec string, opts *options.PushOptions) error {
	runtime := engine.ImageRuntime()
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()
	systemCxt.OCIInsecureSkipTLSVerify = opts.SkipTLSVerify
	systemCxt.DockerInsecureSkipTLSVerify = types.NewOptionalBool(opts.SkipTLSVerify)

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

// AddToManifest :
// for `manifestName`: if it is not exist,will create a new one. if not, it must be an existed manifest name.
// for `imageNameOrIDList`:
// if element is a single image just add it,
// if element is a manifest will add it’s s all instance no matter what platform it is.
func (engine *Engine) AddToManifest(manifestName string, imageNameOrIDList []string, opts *options.ManifestAddOpts) error {
	var (
		runtime = engine.ImageRuntime()
	)

	// check whether manifestName is already existed.
	manifestList, err := runtime.LookupManifestList(manifestName)
	if err == nil {
		return engine.addToManifestList(manifestList, imageNameOrIDList, opts)
	}

	if !errors.Is(err, storage.ErrImageUnknown) {
		return err
	}

	logrus.Infof("will create a new one manifest with name %s", manifestName)
	// if not exit,create a new one
	_, err = engine.CreateManifest(manifestName, &options.ManifestCreateOpts{})
	if err != nil {
		return fmt.Errorf("failed to create a new one manifest with name %s :%v", manifestName, err)
	}
	manifestList, err = runtime.LookupManifestList(manifestName)
	if err != nil {
		return err
	}

	err = engine.addToManifestList(manifestList, imageNameOrIDList, opts)
	if err != nil {
		delErr := engine.DeleteManifests([]string{manifestName}, &options.ManifestDeleteOpts{})
		if delErr != nil {
			return fmt.Errorf("failed to delete %s : %v", manifestName, delErr)
		}
		return err
	}

	return nil
}

func (engine *Engine) addToManifestList(manifestList *libimage.ManifestList, imageNameOrIDList []string, opts *options.ManifestAddOpts) error {
	var (
		imageIDToAdd []string
		err          error
		store        = engine.ImageStore()
	)

	// determine all images
	for _, imageNameOrID := range imageNameOrIDList {
		ret, err := engine.getImageIDList(imageNameOrID)
		if err != nil {
			return fmt.Errorf("failed to look up %s", imageNameOrID)
		}

		imageIDToAdd = append(imageIDToAdd, ret...)
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	// add each to manifest list
	for _, imageID := range imageIDToAdd {
		err = engine.addOneToManifestList(list, imageID, opts)
		if err != nil {
			return fmt.Errorf("failed to add new image %s to manifest :%v ", imageID, err)
		}
	}

	_, err = list.SaveToImage(store, manifestList.ID(), nil, "")

	return err
}

func (engine *Engine) addOneToManifestList(list manifests.List, imageSpec string, opts *options.ManifestAddOpts) error {
	store := engine.ImageStore()
	systemCxt := engine.SystemContext()

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

	logrus.Infof("adding image %s successfully", imageSpec)

	return nil
}

// getImageId get imageID by name Or id,what ever it is an image or a manifest
// if it is image just return imageID
// if it is a manifest, return its included instance IDs.
func (engine *Engine) getImageIDList(imageNameOrID string) ([]string, error) {
	// try to look up `imageNameOrID` as ManifestList
	store := engine.ImageStore()
	img, _, err := engine.ImageRuntime().LookupImage(imageNameOrID, &libimage.LookupImageOptions{
		ManifestList: true,
	})
	if err != nil {
		return nil, err
	}

	isManifest, err := img.IsManifestList(getContext())
	if err != nil {
		return nil, err
	}

	// if not manifest, just return its ID.
	if !isManifest {
		return []string{img.ID()}, nil
	}

	// if it is a manifest, return its included instance ID.
	logrus.Infof("image %q is a manifest list, looking up matching instances", imageNameOrID)

	imageName := img.Names()[0]
	manifestList, err := engine.ImageRuntime().LookupManifestList(imageName)
	if err != nil {
		return nil, err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return nil, err
	}

	var imageIDList []string
	for _, instanceDigest := range list.Instances() {
		images, err := store.ImagesByDigest(instanceDigest)
		if err != nil {
			return nil, err
		}
		if len(images) == 0 {
			return nil, fmt.Errorf("no image matched with digest %s", instanceDigest)
		}
		imageIDList = append(imageIDList, images[0].ID)
	}

	return imageIDList, nil
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
