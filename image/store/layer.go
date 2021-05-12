package store

import (
	"io"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/opencontainers/go-digest"
)

type LayerID digest.Digest

type roLayer struct {
	id LayerID
	size int64
}

func (rl *roLayer) ID() LayerID {
	return rl.id
}

func (rl *roLayer) SimpleID() string {
	return digest.Digest(rl.ID()).Hex()[0:12]
}

func (rl *roLayer) TarStream() (io.ReadCloser, error) {
	id := digest.Digest(rl.id)
	return os.Open(filepath.Join(common.DefaultLayerDBDir, id.Algorithm().String(), id.Hex(), DefaultLayerTarName))
}

func (rl *roLayer) Size() int64 {
	return rl.size
}

func (li LayerID) String() string {
	return string(li)
}

func NewROLayer(LayerDigest digest.Digest, size int64) (Layer, error) {
	err := LayerDigest.Validate()
	if err != nil {
		return nil, err
	}
	return &roLayer{id: LayerID(LayerDigest), size: size}, nil
}