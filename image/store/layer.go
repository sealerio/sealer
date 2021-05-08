package store

import (
	"github.com/alibaba/sealer/common"
	"github.com/opencontainers/go-digest"
	"io"
	"os"
	"path/filepath"
)

type LayerID digest.Digest

type roLayer struct {
	id LayerID
}

func (rl *roLayer) ID() (LayerID, error) {
	lid, err := digest.Parse(rl.id.String())
	if err != nil {
		return "", err
	}
	return LayerID(lid), nil
}

func (rl *roLayer) TarStream() (io.ReadCloser, error) {
	id := digest.Digest(rl.id)
	return os.Open(filepath.Join(common.DefaultLayerDBDir, id.Algorithm().String(), id.Hex(), DefaultLayerTarName))
}

func (li LayerID) String() string {
	return string(li)
}
