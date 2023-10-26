package mfs

import (
	"context"
	"fmt"
	"sync"

	dag "github.com/ipfs/go-merkledag"
	ft "github.com/ipfs/go-unixfs"
	mod "github.com/ipfs/go-unixfs/mod"

	chunker "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipfs/go-ipld-format"
)

// File represents a file in the MFS, its logic its mainly targeted
// to coordinating (potentially many) `FileDescriptor`s pointing to
// it.
type File struct {
	inode

	// Lock to coordinate the `FileDescriptor`s associated to this file.
	desclock sync.RWMutex

	// This isn't any node, it's the root node that represents the
	// entire DAG of nodes that comprise the file.
	// TODO: Rename, there should be an explicit term for these root nodes
	// of a particular sub-DAG that abstract an upper layer's entity.
	node ipld.Node

	// Lock around the `node` that represents this file, necessary because
	// there may be many `FileDescriptor`s operating on this `File`.
	nodeLock sync.RWMutex

	RawLeaves bool
}

// NewFile returns a NewFile object with the given parameters.  If the
// Cid version is non-zero RawLeaves will be enabled.
func NewFile(name string, node ipld.Node, parent parent, dserv ipld.DAGService) (*File, error) {
	fi := &File{
		inode: inode{
			name:       name,
			parent:     parent,
			dagService: dserv,
		},
		node: node,
	}
	if node.Cid().Prefix().Version > 0 {
		fi.RawLeaves = true
	}
	return fi, nil
}

func (fi *File) Open(flags Flags) (_ FileDescriptor, _retErr error) {
	if flags.Write {
		fi.desclock.Lock()
		defer func() {
			if _retErr != nil {
				fi.desclock.Unlock()
			}
		}()
	} else if flags.Read {
		fi.desclock.RLock()
		defer func() {
			if _retErr != nil {
				fi.desclock.RUnlock()
			}
		}()
	} else {
		return nil, fmt.Errorf("file opened for neither reading nor writing")
	}

	fi.nodeLock.RLock()
	node := fi.node
	fi.nodeLock.RUnlock()

	// TODO: Move this `switch` logic outside (maybe even
	// to another package, this seems like a job of UnixFS),
	// `NewDagModifier` uses the IPLD node, we're not
	// extracting anything just doing a safety check.
	switch node := node.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(node.Data())
		if err != nil {
			return nil, err
		}

		switch fsn.Type() {
		default:
			return nil, fmt.Errorf("unsupported fsnode type for 'file'")
		case ft.TSymlink:
			return nil, fmt.Errorf("symlinks not yet supported")
		case ft.TFile, ft.TRaw:
			// OK case
		}
	case *dag.RawNode:
		// Ok as well.
	}

	dmod, err := mod.NewDagModifier(context.TODO(), node, fi.dagService, chunker.DefaultSplitter)
	// TODO: Remove the use of the `chunker` package here, add a new `NewDagModifier` in
	// `go-unixfs` with the `DefaultSplitter` already included.
	if err != nil {
		return nil, err
	}
	dmod.RawLeaves = fi.RawLeaves

	return &fileDescriptor{
		inode: fi,
		flags: flags,
		mod:   dmod,
		state: stateCreated,
	}, nil
}

// Size returns the size of this file
// TODO: Should we be providing this API?
// TODO: There's already a `FileDescriptor.Size()` that
// through the `DagModifier`'s `fileSize` function is doing
// pretty much the same thing as here, we should at least call
// that function and wrap the `ErrNotUnixfs` with an MFS text.
func (fi *File) Size() (int64, error) {
	fi.nodeLock.RLock()
	defer fi.nodeLock.RUnlock()
	switch nd := fi.node.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(nd.Data())
		if err != nil {
			return 0, err
		}
		return int64(fsn.FileSize()), nil
	case *dag.RawNode:
		return int64(len(nd.RawData())), nil
	default:
		return 0, fmt.Errorf("unrecognized node type in mfs/file.Size()")
	}
}

// GetNode returns the dag node associated with this file
// TODO: Use this method and do not access the `nodeLock` directly anywhere else.
func (fi *File) GetNode() (ipld.Node, error) {
	fi.nodeLock.RLock()
	defer fi.nodeLock.RUnlock()
	return fi.node, nil
}

// TODO: Tight coupling with the `FileDescriptor`, at the
// very least this should be an independent function that
// takes a `File` argument and automates the open/flush/close
// operations.
// TODO: Why do we need to flush a file that isn't opened?
// (the `OpenWriteOnly` seems to implicitly be targeting a
// closed file, a file we forgot to flush? can we close
// a file without flushing?)
func (fi *File) Flush() error {
	// open the file in fullsync mode
	fd, err := fi.Open(Flags{Write: true, Sync: true})
	if err != nil {
		return err
	}

	defer fd.Close()

	return fd.Flush()
}

func (fi *File) Sync() error {
	// just being able to take the writelock means the descriptor is synced
	// TODO: Why?
	fi.desclock.Lock()
	defer fi.desclock.Unlock() // Defer works around "empty critical section (SA2001)"
	return nil
}

// Type returns the type FSNode this is
func (fi *File) Type() NodeType {
	return TFile
}
