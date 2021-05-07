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
	SimpleID() string
	TarStream() (io.ReadCloser, error)
}
