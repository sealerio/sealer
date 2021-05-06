package store

import (
	"io"
)

type LayerStore interface {
	Get(id LayerID) Layer
	RegisterLayerIfNotPresent(closer io.ReadCloser, id LayerID) error
}

type Layer interface {
	ID() (LayerID, error)
	TarStream() (io.ReadCloser, error)
}
