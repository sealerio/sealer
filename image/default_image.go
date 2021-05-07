package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/registry"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/compress"
	"github.com/alibaba/sealer/utils/progress"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

const (
	imagePullComplete  = "Pull Completed"
	imageDownloading   = "Downloading"
	imageExtracting    = "Extracting"
	imagePushing       = "Pushing"
	imagePushCompleted = "Push Completed"
	imageCompressing   = "Compressing"
)

// DefaultImageService is the default service, which is used for image pull/push
type DefaultImageService struct {
	BaseImageManager
}

// PullIfNotExist is used to pull image if not exists locally
func (d DefaultImageService) PullIfNotExist(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	_, err = imageutils.GetImage(named.Raw())
	if err == nil {
		logger.Info("image %s already exists", named.Raw())
		return nil
	}

	return d.Pull(imageName)
}

// Pull always do pull action
func (d DefaultImageService) Pull(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	err = d.initRegistry(named.Domain())
	if err != nil {
		return err
	}

	image, err := d.remoteImage(named.Raw())
	if err != nil {
		return err
	}
	// TODO rely on id next
	image.Name = named.Raw()
	fmt.Printf("Start to Pull Image %s \n", named.Raw())
	return d.pull(*image)
}

// Push push local image to remote registry
func (d DefaultImageService) Push(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	err = d.initRegistry(named.Domain())
	if err != nil {
		return err
	}

	image, err := imageutils.GetImage(named.Raw())
	if err != nil {
		return err
	}

	fmt.Printf("Start to Push Image %s \n", named.Raw())
	descriptors, err := d.pushLayers(named, image)
	if err != nil {
		return err
	}

	metadataBytes, err := d.pushManifestConfig(named, *image)
	if err != nil {
		return err
	}

	return d.pushManifest(metadataBytes, named, descriptors)
}

// Login login into a registry, for saving auth info in ~/.docker/config.json
func (d DefaultImageService) Login(RegistryURL, RegistryUsername, RegistryPasswd string) error {
	_, err := registry.New(context.Background(), types.AuthConfig{ServerAddress: RegistryURL, Username: RegistryUsername, Password: RegistryPasswd}, registry.Opt{Insecure: true, Debug: true})
	if err != nil {
		logger.Error("%v authentication failed", RegistryURL)
		return err
	}
	if err := utils.SetDockerConfig(RegistryURL, RegistryUsername, RegistryPasswd); err != nil {
		return err
	}
	logger.Info("%s login %s success", RegistryUsername, RegistryURL)
	return nil
}

//func (d DefaultImageService) Load(imageSrc string) error {
//	panic("implement me")
//}

//func (d DefaultImageService) Save(imageName string, imageTar string) error {
//	will be accomplished
//	img, err := localImage(imageName)
//	if err != nil {
//		return err
//	}
//
//	tarFile, err := os.OpenFile(imageTar, os.O_CREATE|os.O_TRUNC, 0766)
//	if err != nil {
//		return err
//	}
//
//	for _, layer := range img.Spec.Layers {
//		compress.Compress("", layer.Hash)
//		io.Copy(tarFile)
//	}
//	compress.Compress()
//	panic("implement me")
//}

//func (d DefaultImageService) Merge(image *v1.Image) (err error) {
//	var layers []string
//	// TODO merge baseImage layers
//	for _, l := range image.Spec.Layers {
//		if l.Type == common.COPYCOMMAND {
//			layers = append(layers, fmt.Sprintf("%s/%s", common.DefaultImageRootDir, l.Hash))
//		}
//	}
//
//	driver := mount.NewMountDriver()
//	err = driver.Mount("", "", layers...)
//	return err
//}

func (d DefaultImageService) downloadLayers(named reference.Named, manifest schema2.Manifest) (err error) {
	flow := progress.NewProgressFlow()
	errorCh := make(chan error, 2*len(manifest.Layers))
	defer func() {
		close(errorCh)
		lerr := errors.New("failed to upload layers")
		for e := range errorCh {
			err = errors.Wrap(e, lerr.Error())
			lerr = err
		}
	}()

	for _, layer := range manifest.Layers {
		hex := layer.Digest.Hex()
		shortHex := hex
		if len(shortHex) > 12 {
			shortHex = shortHex[0:12]
		}
		// check if the layer exists locally
		if _, err := os.Stat(filepath.Join(common.DefaultLayerDir, hex)); err != nil {
			if !os.IsNotExist(err) {
				logger.Error(err)
				errorCh <- err
				continue
			}
		} else {
			flow.ShowMessage(shortHex+" already exists", nil)
			continue
		}

		// get layers stream first
		blobReader, err := d.registry.DownloadLayer(context.Background(), named.Repo(), layer.Digest)
		if err != nil {
			flow.ShowMessage(shortHex+fmt.Sprintf(" failed to pull layer, err: %s", err), nil)
			errorCh <- err
			continue
		}

		flow.AddProgressTasks(progress.TaskDef{
			Task:       hex[0:12],
			Job:        imageDownloading + "&" + imageExtracting,
			Max:        layer.Size,
			SuccessMsg: shortHex + " " + imagePullComplete,
			ProgressSrc: progress.TakeOverTask{
				Cxt: progress.Context{}.WithReader(blobReader),
				Action: func(cxt progress.Context) error {
					rc := cxt.GetCurrentReaderCloser()
					if rc == nil {
						err = errors.New("failed to start uploading layer, err: no reader found")
						errorCh <- err
						return err
					}
					defer rc.Close()
					curBar := cxt.GetCurrentBar()
					if curBar == nil {
						err = errors.New("failed to start uploading layer, err: no current bar found")
						errorCh <- err
						return err
					}

					if err := compress.Uncompress(curBar.ProxyReader(rc), filepath.Join(common.DefaultLayerDir, hex)); err != nil {
						errorCh <- err
						return err
					}
					return nil
				},
			},
		})
	}

	flow.Start()
	return nil
}

func (d DefaultImageService) uploadLayers(repo string, layers []v1.Layer, blobs chan distribution.Descriptor) (err error) {
	flow := progress.NewProgressFlow()
	errCh := make(chan error, 2*len(layers))
	defer func() {
		close(errCh)
		lerr := errors.New("failed to upload layers")
		for e := range errCh {
			err = errors.Wrap(e, lerr.Error())
			lerr = err
		}
	}()

	for _, layer := range layers {
		// do not push empty layer
		if layer.Hash == "" {
			continue
		}

		shortHex := layer.Hash
		if len(shortHex) > 12 {
			shortHex = shortHex[0:12]
		}
		// check if the layer exists
		layerDig := digest.NewDigestFromEncoded(digest.SHA256, layer.Hash)
		// TODO next we need to know the err type, 404 or sth else
		blob, err := d.registry.LayerMetadata(repo, layerDig)
		if err == nil {
			blobs <- buildBlobs(layerDig, blob.Size, schema2.MediaTypeLayer)
			flow.ShowMessage(shortHex+" "+"already exist remotely", nil)
			continue
		}

		barID := utils.GenUniqueID(8)
		flow.AddProgressTasks(progress.TaskDef{
			Task: shortHex,
			Job:  imageCompressing,
			Max:  1,
			ID:   barID,
			ProgressSrc: progress.TakeOverTask{
				Cxt: progress.Context{},
				Action: func(cxt progress.Context) error {
					var file *os.File
					defer func() {
						//file compress failed, clean file
						if err != nil {
							utils.CleanFile(file)
						}
					}()

					if file, err = compress.Compress(filepath.Join(common.DefaultLayerDir, layer.Hash), "", nil); err != nil {
						errCh <- err
						return err
					}
					// pass to next progress task
					cxt.WithReader(file)
					return nil
				},
			},
		})

		flow.AddProgressTasks(progress.TaskDef{
			Task:       shortHex,
			Job:        imagePushing,
			Max:        1,
			ID:         barID,
			SuccessMsg: shortHex + " " + imagePushCompleted,
			FailMsg:    shortHex,
			ProgressSrc: progress.TakeOverTask{
				Cxt: progress.Context{},
				Action: func(cxt progress.Context) error {
					var file *os.File
					file, ok := cxt.GetCurrentReaderCloser().(*os.File)
					if !ok || file == nil {
						err := errors.New("failed to start uploading layer, err: no reader found or reader is not file")
						errCh <- err
						return err
					}
					defer utils.CleanFile(file)
					if _, err = file.Seek(0, 0); err != nil {
						errCh <- err
						return err
					}
					fi, err := file.Stat()
					if err != nil {
						errCh <- err
						return err
					}
					curBar := cxt.GetCurrentBar()
					if curBar == nil {
						err = errors.New("failed to start uploading layer, err: no current bar found")
						errCh <- err
						return err
					}
					// there is no better way, we can't know file size on registering the upload process bar
					// so we can set the total of the bar at the time only
					curBar.SetTotal(fi.Size(), false)
					prc := curBar.ProxyReader(file)
					if err := d.registry.UploadLayer(context.Background(), repo, layerDig, prc); err != nil {
						errCh <- err
						return err
					}
					blobs <- buildBlobs(layerDig, fi.Size(), schema2.MediaTypeLayer)
					return nil
				},
			},
		})
	}
	flow.Start()
	return
}

func (d DefaultImageService) uploadImageMetadata(repo string, image v1.Image) ([]byte, error) {
	byts, err := json.Marshal(image)
	if err != nil {
		return nil, err
	}

	dig := digest.FromBytes(byts)
	err = d.registry.UploadLayer(context.Background(), repo, dig, bytes.NewReader(byts))
	if err != nil {
		return nil, err
	}

	return byts, nil
}

func (d DefaultImageService) remoteImage(imageName string) (*v1.Image, error) {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return nil, err
	}
	manifest, err := d.registry.ManifestV2(context.Background(), named.Repo(), named.Tag())
	if err != nil {
		return nil, err
	}

	remoteImage, err := d.downloadImageManifestConfig(named, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}

	return &remoteImage, nil
}

func (d DefaultImageService) pull(img v1.Image) error {
	named, err := reference.ParseToNamed(img.Name)
	if err != nil {
		return err
	}

	repo, tag := named.Repo(), named.Tag()
	manifest, err := d.registry.ManifestV2(context.Background(), repo, tag)
	if err != nil {
		return err
	}

	err = d.downloadLayers(named, manifest)
	if err != nil {
		return err
	}

	return d.syncImageLocal(img)
}

func (d DefaultImageService) pushLayers(named reference.Named, image *v1.Image) ([]distribution.Descriptor, error) {
	if len(image.Spec.Layers) == 0 {
		return []distribution.Descriptor{}, errors.New(fmt.Sprintf("image %s layers empty", named.Raw()))
	}

	var descriptors []distribution.Descriptor
	descriptorsCh := make(chan distribution.Descriptor, len(image.Spec.Layers))
	err := d.uploadLayers(named.Repo(), image.Spec.Layers, descriptorsCh)
	close(descriptorsCh)
	if err != nil {
		return descriptors, err
	}

	for des := range descriptorsCh {
		descriptors = append(descriptors, des)
	}

	return descriptors, nil
}

func (d DefaultImageService) pushManifestConfig(named reference.Named, image v1.Image) ([]byte, error) {
	// save image json data as manifests config
	return d.uploadImageMetadata(
		named.Repo(),
		image,
	)
}

func (d DefaultImageService) pushManifest(metadata []byte, named reference.Named, descriptors []distribution.Descriptor) error {
	bs := &blobService{descriptors: make(map[digest.Digest]distribution.Descriptor)}
	mBuilder := schema2.NewManifestBuilder(bs, schema2.MediaTypeManifest, metadata)
	for _, b := range descriptors {
		err := mBuilder.AppendReference(b)
		if err != nil {
			return err
		}
	}

	built, err := mBuilder.Build(context.Background())
	if err != nil {
		return err
	}

	return d.registry.PutManifest(context.Background(), named.Repo(), named.Tag(), built)
}
