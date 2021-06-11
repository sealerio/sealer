package store

import (
	"sync"

	"github.com/alibaba/sealer/logger"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/opencontainers/go-digest"
)

var imagestore *store
var once sync.Once

//StoreID is caculated from a series of serialized layers. The layers hash
//value is "", but the layers value and type is not empty. It is used in
//image store to help to reference to a stack of layers with a single identifier.
type StoreID digest.Digest

func (id StoreID) String() string {
	return id.Digest().String()
}

// Digest converts ID into a digest
func (id StoreID) Digest() digest.Digest {
	return digest.Digest(id)
}

type ImageID digest.Digest

func (id ImageID) String() string {
	return id.Digest().String()
}

// Digest converts ID into a digest
func (id ImageID) Digest() digest.Digest {
	return digest.Digest(id)
}

// Store is an interface for manipulating images
type Store interface {
	Images() map[ImageID]*v1.Image
	SetParent(id StoreID, parent StoreID) error
	GetParent(id StoreID) (StoreID, error)
	GetLayer(id StoreID) (v1.Layer, error)
	Children(id StoreID) []StoreID
}

type imageMeta struct {
	layer v1.Layer
	// use store id
	parent StoreID
	// use store id
	children map[StoreID]struct{}
}

type store struct {
	sync.RWMutex
	// use store id
	images map[StoreID]*imageMeta
	fs     StoreBackend
	ls     LayerStore
}

func NewImageStore(fs StoreBackend, ls LayerStore) (Store, error) {
	once.Do(func() {
		imagestore = &store{
			images: make(map[StoreID]*imageMeta),
			fs:     fs,
			ls:     ls,
		}

		if err := imagestore.restore(); err != nil {
			return
		}
	})
	return imagestore, nil
}

// restore reads all images saved in filesystem and calculate their storeid
func (is *store) restore() error {
	is.Lock()
	defer is.Unlock()

	//read all image layers
	images := is.Images()
	for _, image := range images {
		layers := image.Spec.Layers
		var tempLayers []v1.Layer
		for i, layer := range layers {
			tempLayers = append(tempLayers, v1.Layer{Value: layer.Value, Type: layer.Type})
			image.Spec.Layers = tempLayers
			// at present, we only calculate layers without hash
			storeid, err := GetStoreIDFromImage(*image)
			if err != nil {
				logger.Error(err)
				break
			}

			imagemeta, ok := is.images[storeid]
			if !ok {
				imagemeta = &imageMeta{
					layer:    layer,
					children: make(map[StoreID]struct{}),
				}
				is.images[storeid] = imagemeta
			}

			if i < len(layers)-1 {
				image.Spec.Layers = append(tempLayers, v1.Layer{Value: layers[i+1].Value, Type: layers[i+1].Type})
				childStoreID, err := GetStoreIDFromImage(*image)
				if err != nil {
					logger.Error(err)
					break
				}
				imagemeta.children[childStoreID] = struct{}{}
				_, ok := is.images[childStoreID]
				if !ok {
					is.images[childStoreID] = &imageMeta{
						layer:    layers[i+1],
						parent:   storeid,
						children: make(map[StoreID]struct{}),
					}
				}
			}
		}
	}

	return nil
}

func (is *store) SetParent(id StoreID, parent StoreID) error {
	var (
		parentMeta *imageMeta
		imagemeta  *imageMeta
		ok         bool
	)

	parentMeta, ok = is.images[parent]
	if !ok {
		return errors.Errorf("unknown parent store ID %s", parent.String())
	}
	imagemeta, ok = is.images[id]
	if !ok {
		return errors.Errorf("unknown store ID %s", id.String())
	}
	if parent, err := is.GetParent(id); err == nil {
		delete(is.images[parent].children, id)
	}
	parentMeta.children[id] = struct{}{}
	imagemeta.parent = parent
	return nil
}

func (is *store) GetParent(id StoreID) (StoreID, error) {
	var (
		imagemeta *imageMeta
		ok        bool
	)
	imagemeta, ok = is.images[id]
	if !ok {
		return "", errors.Errorf("unknown store ID %s", id.String())
	}

	// If the parent id of id is "", we return error
	if imagemeta.parent == "" {
		return "", errors.Errorf("store ID %s has no parent", id.String())
	}

	return imagemeta.parent, nil
}

func (is *store) GetLayer(id StoreID) (v1.Layer, error) {
	is.RLock()
	defer is.RUnlock()

	if imagemeta, ok := is.images[id]; ok {
		return imagemeta.layer, nil
	}

	return v1.Layer{}, errors.Errorf("no layer for store id %s in file system", id)
}

func (is *store) Children(storeID StoreID) []StoreID {
	var (
		ids       []StoreID
		ok        bool
		imagemeta *imageMeta
	)

	is.Lock()
	defer is.Unlock()

	if imagemeta, ok = is.images[storeID]; ok {
		for tempid := range imagemeta.children {
			ids = append(ids, tempid)
		}
	}

	return ids
}

func (is *store) Images() map[ImageID]*v1.Image {
	var (
		images  map[ImageID]*v1.Image
		configs [][]byte
		err     error
	)

	images = make(map[ImageID]*v1.Image)
	configs, err = is.fs.List()
	if err != nil {
		logger.Error("failed to get images from file system, err: %v", err)
		return nil
	}
	for _, config := range configs {
		img := &v1.Image{}
		err = yaml.Unmarshal(config, img)
		if err != nil {
			logger.Error("failed to unmarshal bytes into image")
			continue
		}
		dgst := digest.FromBytes(config)
		images[ImageID(dgst)] = img
	}

	return images
}

func GetStoreIDFromImage(image v1.Image) (StoreID, error) {
	var (
		config []byte
		err    error
	)

	for _, layer := range image.Spec.Layers {
		layer.Hash = ""
	}

	config, err = yaml.Marshal(image.Spec.Layers)
	if err != nil {
		return "", errors.Errorf("failed to marshal image %s into bytes, err: %v",
			image.Spec.ID, err)
	}

	return StoreID(digest.FromBytes(config)), nil
}
