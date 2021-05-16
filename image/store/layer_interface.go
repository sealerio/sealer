package store

import (
	"io"
)

type LayerStore interface {
	Get(id LayerID) Layer
	RegisterLayerIfNotPresent(layer Layer) error
	Delete(id LayerID) error
}

type Layer interface {
	ID() LayerID
	SimpleID() string
	TarStream() (io.ReadCloser, error)
	Size() int64
}
