package mfs

import (
	"fmt"
	"io"

	mod "github.com/ipfs/go-unixfs/mod"

	context "context"

	ipld "github.com/ipfs/go-ipld-format"
)

type state uint8

const (
	stateCreated state = iota
	stateFlushed
	stateDirty
	stateClosed
)

// One `File` can have many `FileDescriptor`s associated to it
// (only one if it's RW, many if they are RO, see `File.desclock`).
// A `FileDescriptor` contains the "view" of the file (through an
// instance of a `DagModifier`), that's why it (and not the `File`)
// has the responsibility to `Flush` (which crystallizes that view
// in the `File`'s `Node`).
type FileDescriptor interface {
	io.Reader
	CtxReadFull(context.Context, []byte) (int, error)

	io.Writer
	io.WriterAt

	io.Closer
	io.Seeker

	Truncate(int64) error
	Size() (int64, error)
	Flush() error
}

type fileDescriptor struct {
	inode *File
	mod   *mod.DagModifier
	flags Flags

	state state
}

func (fi *fileDescriptor) checkWrite() error {
	if fi.state == stateClosed {
		return ErrClosed
	}
	if !fi.flags.Write {
		return fmt.Errorf("file is read-only")
	}
	return nil
}

func (fi *fileDescriptor) checkRead() error {
	if fi.state == stateClosed {
		return ErrClosed
	}
	if !fi.flags.Read {
		return fmt.Errorf("file is write-only")
	}
	return nil
}

// Size returns the size of the file referred to by this descriptor
func (fi *fileDescriptor) Size() (int64, error) {
	return fi.mod.Size()
}

// Truncate truncates the file to size
func (fi *fileDescriptor) Truncate(size int64) error {
	if err := fi.checkWrite(); err != nil {
		return fmt.Errorf("truncate failed: %s", err)
	}
	fi.state = stateDirty
	return fi.mod.Truncate(size)
}

// Write writes the given data to the file at its current offset
func (fi *fileDescriptor) Write(b []byte) (int, error) {
	if err := fi.checkWrite(); err != nil {
		return 0, fmt.Errorf("write failed: %s", err)
	}
	fi.state = stateDirty
	return fi.mod.Write(b)
}

// Read reads into the given buffer from the current offset
func (fi *fileDescriptor) Read(b []byte) (int, error) {
	if err := fi.checkRead(); err != nil {
		return 0, fmt.Errorf("read failed: %s", err)
	}
	return fi.mod.Read(b)
}

// Read reads into the given buffer from the current offset
func (fi *fileDescriptor) CtxReadFull(ctx context.Context, b []byte) (int, error) {
	if err := fi.checkRead(); err != nil {
		return 0, fmt.Errorf("read failed: %s", err)
	}
	return fi.mod.CtxReadFull(ctx, b)
}

// Close flushes, then propogates the modified dag node up the directory structure
// and signals a republish to occur
func (fi *fileDescriptor) Close() error {
	if fi.state == stateClosed {
		return ErrClosed
	}
	if fi.flags.Write {
		defer fi.inode.desclock.Unlock()
	} else if fi.flags.Read {
		defer fi.inode.desclock.RUnlock()
	}
	err := fi.flushUp(fi.flags.Sync)
	fi.state = stateClosed
	return err
}

// Flush generates a new version of the node of the underlying
// UnixFS directory (adding it to the DAG service) and updates
// the entry in the parent directory (setting `fullSync` to
// propagate the update all the way to the root).
func (fi *fileDescriptor) Flush() error {
	return fi.flushUp(true)
}

// flushUp syncs the file and adds it to the dagservice
// it *must* be called with the File's lock taken
// If `fullSync` is set the changes are propagated upwards
// (the `Up` part of `flushUp`).
func (fi *fileDescriptor) flushUp(fullSync bool) error {
	var nd ipld.Node
	switch fi.state {
	case stateCreated, stateDirty:
		var err error
		nd, err = fi.mod.GetNode()
		if err != nil {
			return err
		}
		err = fi.inode.dagService.Add(context.TODO(), nd)
		if err != nil {
			return err
		}

		// TODO: Very similar logic to the update process in
		// `Directory`, the logic should be unified, both structures
		// (`File` and `Directory`) are backed by a IPLD node with
		// a UnixFS format that is the actual target of the update
		// (regenerating it and adding it to the DAG service).
		fi.inode.nodeLock.Lock()
		// Always update the file descriptor's inode with the created/modified node.
		fi.inode.node = nd
		// Save the members to be used for subsequent calls
		parent := fi.inode.parent
		name := fi.inode.name
		fi.inode.nodeLock.Unlock()

		// Bubble up the update's to the parent, only if fullSync is set to true.
		if fullSync {
			if err := parent.updateChildEntry(child{name, nd}); err != nil {
				return err
			}
		}

		fi.state = stateFlushed
		return nil
	case stateFlushed:
		return nil
	default:
		panic("invalid state")
	}
}

// Seek implements io.Seeker
func (fi *fileDescriptor) Seek(offset int64, whence int) (int64, error) {
	if fi.state == stateClosed {
		return 0, fmt.Errorf("seek failed: %s", ErrClosed)
	}
	return fi.mod.Seek(offset, whence)
}

// Write At writes the given bytes at the offset 'at'
func (fi *fileDescriptor) WriteAt(b []byte, at int64) (int, error) {
	if err := fi.checkWrite(); err != nil {
		return 0, fmt.Errorf("write-at failed: %s", err)
	}
	fi.state = stateDirty
	return fi.mod.WriteAt(b, at)
}
