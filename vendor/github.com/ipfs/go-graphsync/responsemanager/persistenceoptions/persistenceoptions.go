package persistenceoptions

import (
	"errors"
	"sync"

	"github.com/ipld/go-ipld-prime"
)

// PersistenceOptions is a registry of loaders for persistence options
type PersistenceOptions struct {
	persistenceOptionsLk sync.RWMutex
	persistenceOptions   map[string]ipld.LinkSystem
}

// New returns a new registry of persistence options
func New() *PersistenceOptions {
	return &PersistenceOptions{
		persistenceOptions: make(map[string]ipld.LinkSystem),
	}
}

// Register registers a new link system for the response manager
func (po *PersistenceOptions) Register(name string, linkSystem ipld.LinkSystem) error {
	po.persistenceOptionsLk.Lock()
	defer po.persistenceOptionsLk.Unlock()
	_, ok := po.persistenceOptions[name]
	if ok {
		return errors.New("persistence option alreayd registered")
	}
	po.persistenceOptions[name] = linkSystem
	return nil
}

// Unregister unregisters a link system for the response manager
func (po *PersistenceOptions) Unregister(name string) error {
	po.persistenceOptionsLk.Lock()
	defer po.persistenceOptionsLk.Unlock()
	_, ok := po.persistenceOptions[name]
	if !ok {
		return errors.New("persistence option is not registered")
	}
	delete(po.persistenceOptions, name)
	return nil
}

// GetLinkSystem returns the link system for the named persistence option
func (po *PersistenceOptions) GetLinkSystem(name string) (ipld.LinkSystem, bool) {
	po.persistenceOptionsLk.RLock()
	defer po.persistenceOptionsLk.RUnlock()
	linkSystem, ok := po.persistenceOptions[name]
	return linkSystem, ok
}
