package io

import (
	"bytes"
	"context"
	"errors"
	"io"

	ipld "github.com/ipfs/go-ipld-format"
	mdag "github.com/ipfs/go-merkledag"
	unixfs "github.com/ipfs/go-unixfs"
)

// Common errors
var (
	ErrIsDir            = errors.New("this dag node is a directory")
	ErrCantReadSymlinks = errors.New("cannot currently read symlinks")
	ErrUnkownNodeType   = errors.New("unknown node type")
	ErrSeekNotSupported = errors.New("file does not support seeking")
)

// TODO: Rename the `DagReader` interface, this doesn't read *any* DAG, just
// DAGs with UnixFS node (and it *belongs* to the `unixfs` package). Some
// alternatives: `FileReader`, `UnixFSFileReader`, `UnixFSReader`.

// A DagReader provides read-only read and seek acess to a unixfs file.
// Different implementations of readers are used for the different
// types of unixfs/protobuf-encoded nodes.
type DagReader interface {
	ReadSeekCloser
	Size() uint64
	CtxReadFull(context.Context, []byte) (int, error)
}

// A ReadSeekCloser implements interfaces to read, copy, seek and close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
	io.WriterTo
}

// NewDagReader creates a new reader object that reads the data represented by
// the given node, using the passed in DAGService for data retrieval.
func NewDagReader(ctx context.Context, n ipld.Node, serv ipld.NodeGetter) (DagReader, error) {
	var size uint64

	switch n := n.(type) {
	case *mdag.RawNode:
		size = uint64(len(n.RawData()))

	case *mdag.ProtoNode:
		fsNode, err := unixfs.FSNodeFromBytes(n.Data())
		if err != nil {
			return nil, err
		}

		switch fsNode.Type() {
		case unixfs.TFile, unixfs.TRaw:
			size = fsNode.FileSize()

		case unixfs.TDirectory, unixfs.THAMTShard:
			// Dont allow reading directories
			return nil, ErrIsDir

		case unixfs.TMetadata:
			if len(n.Links()) == 0 {
				return nil, errors.New("incorrectly formatted metadata object")
			}
			child, err := n.Links()[0].GetNode(ctx, serv)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*mdag.ProtoNode)
			if !ok {
				return nil, mdag.ErrNotProtobuf
			}
			return NewDagReader(ctx, childpb, serv)
		case unixfs.TSymlink:
			return nil, ErrCantReadSymlinks
		default:
			return nil, unixfs.ErrUnrecognizedType
		}
	default:
		return nil, ErrUnkownNodeType
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)

	return &dagReader{
		ctx:       ctxWithCancel,
		cancel:    cancel,
		serv:      serv,
		size:      size,
		rootNode:  n,
		dagWalker: ipld.NewWalker(ctxWithCancel, ipld.NewNavigableIPLDNode(n, serv)),
	}, nil
}

// dagReader provides a way to easily read the data contained in a dag.
type dagReader struct {

	// Structure to perform the DAG iteration and search, the reader
	// just needs to add logic to the `Visitor` callback passed to
	// `Iterate` and `Seek`.
	dagWalker *ipld.Walker

	// Buffer with the data extracted from the current node being visited.
	// To avoid revisiting a node to complete a (potential) partial read
	// (or read after seek) the node's data is fully extracted in a single
	// `readNodeDataBuffer` operation.
	currentNodeData *bytes.Reader

	// Implements the `Size()` API.
	size uint64

	// Current offset for the read head within the DAG file.
	offset int64

	// Root node of the DAG, stored to re-create the `dagWalker` (effectively
	// re-setting the position of the reader, used during `Seek`).
	rootNode ipld.Node

	// Context passed to the `dagWalker`, the `cancel` function is used to
	// cancel read operations (cancelling requested child node promises,
	// see `ipld.NavigableIPLDNode.FetchChild` for details).
	ctx    context.Context
	cancel func()

	// Passed to the `dagWalker` that will use it to request nodes.
	// TODO: Revisit name.
	serv ipld.NodeGetter
}

// Size returns the total size of the data from the DAG structured file.
func (dr *dagReader) Size() uint64 {
	return dr.size
}

// Read implements the `io.Reader` interface through the `CtxReadFull`
// method using the DAG reader's internal context.
func (dr *dagReader) Read(b []byte) (int, error) {
	return dr.CtxReadFull(dr.ctx, b)
}

// CtxReadFull reads data from the DAG structured file. It always
// attempts a full read of the DAG until the `out` buffer is full.
// It uses the `Walker` structure to iterate the file DAG and read
// every node's data into the `out` buffer.
func (dr *dagReader) CtxReadFull(ctx context.Context, out []byte) (n int, err error) {
	// Set the `dagWalker`'s context to the `ctx` argument, it will be used
	// to fetch the child node promises (see
	// `ipld.NavigableIPLDNode.FetchChild` for details).
	dr.dagWalker.SetContext(ctx)

	// If there was a partially read buffer from the last visited
	// node read it before visiting a new one.
	if dr.currentNodeData != nil {
		// TODO: Move this check inside `readNodeDataBuffer`?
		n = dr.readNodeDataBuffer(out)

		if n == len(out) {
			return n, nil
			// Output buffer full, no need to traverse the DAG.
		}
	}

	// Iterate the DAG calling the passed `Visitor` function on every node
	// to read its data into the `out` buffer, stop if there is an error or
	// if the entire DAG is traversed (`EndOfDag`).
	err = dr.dagWalker.Iterate(func(visitedNode ipld.NavigableNode) error {
		node := ipld.ExtractIPLDNode(visitedNode)

		// Skip internal nodes, they shouldn't have any file data
		// (see the `balanced` package for more details).
		if len(node.Links()) > 0 {
			return nil
		}

		err = dr.saveNodeData(node)
		if err != nil {
			return err
		}
		// Save the leaf node file data in a buffer in case it is only
		// partially read now and future `CtxReadFull` calls reclaim the
		// rest (as each node is visited only once during `Iterate`).
		//
		// TODO: We could check if the entire node's data can fit in the
		// remaining `out` buffer free space to skip this intermediary step.

		n += dr.readNodeDataBuffer(out[n:])

		if n == len(out) {
			// Output buffer full, no need to keep traversing the DAG,
			// signal the `Walker` to pause the iteration.
			dr.dagWalker.Pause()
		}

		return nil
	})

	if err == ipld.EndOfDag {
		return n, io.EOF
		// Reached the end of the (DAG) file, no more data to read.
	} else if err != nil {
		return n, err
		// Pass along any other errors from the `Visitor`.
	}

	return n, nil
}

// Save the UnixFS `node`'s data into the internal `currentNodeData` buffer to
// later move it to the output buffer (`Read`) or seek into it (`Seek`).
func (dr *dagReader) saveNodeData(node ipld.Node) error {
	extractedNodeData, err := unixfs.ReadUnixFSNodeData(node)
	if err != nil {
		return err
	}

	dr.currentNodeData = bytes.NewReader(extractedNodeData)
	return nil
}

// Read the `currentNodeData` buffer into `out`. This function can't have
// any errors as it's always reading from a `bytes.Reader` and asking only
// the available data in it.
func (dr *dagReader) readNodeDataBuffer(out []byte) int {

	n, _ := dr.currentNodeData.Read(out)
	// Ignore the error as the EOF may not be returned in the first
	// `Read` call, explicitly ask for an empty buffer below to check
	// if we've reached the end.

	if dr.currentNodeData.Len() == 0 {
		dr.currentNodeData = nil
		// Signal that the buffer was consumed (for later `Read` calls).
		// This shouldn't return an EOF error as it's just the end of a
		// single node's data, not the entire DAG.
	}

	dr.offset += int64(n)
	// TODO: Should `offset` be incremented here or in the calling function?
	// (Doing it here saves LoC but may be confusing as it's more hidden).

	return n
}

// Similar to `readNodeDataBuffer` but it writes the contents to
// an `io.Writer` argument.
//
// TODO: Check what part of the logic between the two functions
// can be extracted away.
func (dr *dagReader) writeNodeDataBuffer(w io.Writer) (int64, error) {

	n, err := dr.currentNodeData.WriteTo(w)
	if err != nil {
		return n, err
	}

	if dr.currentNodeData.Len() == 0 {
		dr.currentNodeData = nil
		// Signal that the buffer was consumed (for later `Read` calls).
		// This shouldn't return an EOF error as it's just the end of a
		// single node's data, not the entire DAG.
	}

	dr.offset += int64(n)
	return n, nil
}

// WriteTo writes to the given writer.
// This follows the `bytes.Reader.WriteTo` implementation
// where it starts from the internal index that may have
// been modified by other `Read` calls.
//
// TODO: This implementation is very similar to `CtxReadFull`,
// the common parts should be abstracted away.
func (dr *dagReader) WriteTo(w io.Writer) (n int64, err error) {
	// Use the internal reader's context to fetch the child node promises
	// (see `ipld.NavigableIPLDNode.FetchChild` for details).
	dr.dagWalker.SetContext(dr.ctx)

	// If there was a partially read buffer from the last visited
	// node read it before visiting a new one.
	if dr.currentNodeData != nil {
		n, err = dr.writeNodeDataBuffer(w)
		if err != nil {
			return n, err
		}
	}

	// Iterate the DAG calling the passed `Visitor` function on every node
	// to read its data into the `out` buffer, stop if there is an error or
	// if the entire DAG is traversed (`EndOfDag`).
	err = dr.dagWalker.Iterate(func(visitedNode ipld.NavigableNode) error {
		node := ipld.ExtractIPLDNode(visitedNode)

		// Skip internal nodes, they shouldn't have any file data
		// (see the `balanced` package for more details).
		if len(node.Links()) > 0 {
			return nil
		}

		err = dr.saveNodeData(node)
		if err != nil {
			return err
		}
		// Save the leaf node file data in a buffer in case it is only
		// partially read now and future `CtxReadFull` calls reclaim the
		// rest (as each node is visited only once during `Iterate`).

		written, err := dr.writeNodeDataBuffer(w)
		n += written
		if err != nil {
			return err
		}

		return nil
	})

	if err == ipld.EndOfDag {
		return n, nil
	}

	return n, err
}

// Close the reader (cancelling fetch node operations requested with
// the internal context, that is, `Read` calls but not `CtxReadFull`
// with user-supplied contexts).
func (dr *dagReader) Close() error {
	dr.cancel()
	return nil
}

// Seek implements `io.Seeker` seeking to a given offset in the DAG file,
// it matches the standard unix `seek`. It moves the position of the internal
// `dagWalker` and may also leave a `currentNodeData` buffer loaded in case
// the seek is performed to the middle of the data in a node.
//
// TODO: Support seeking from the current position (relative seek)
// through the `dagWalker` in `io.SeekCurrent`.
func (dr *dagReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return dr.offset, errors.New("invalid offset")
		}

		if offset == dr.offset {
			return offset, nil
			// Already at the requested `offset`, nothing to do.
		}

		left := offset
		// Amount left to seek.

		// Seek from the beginning of the DAG.
		dr.resetPosition()

		// Shortcut seeking to the beginning, we're already there.
		if offset == 0 {
			return 0, nil
		}

		// Use the internal reader's context to fetch the child node promises
		// (see `ipld.NavigableIPLDNode.FetchChild` for details).
		dr.dagWalker.SetContext(dr.ctx)
		// TODO: Performance: we could adjust here `preloadSize` of
		// `ipld.NavigableIPLDNode` also, when seeking we only want
		// to fetch one child at a time.

		// Seek the DAG by calling the provided `Visitor` function on every
		// node the `dagWalker` descends to while searching which can be
		// either an internal or leaf node. In the internal node case, check
		// the child node sizes and set the corresponding child index to go
		// down to next. In the leaf case (last visit of the search), if there
		// is still an amount `left` to seek do it inside the node's data
		// saved in the `currentNodeData` buffer, leaving it ready for a `Read`
		// call.
		err := dr.dagWalker.Seek(func(visitedNode ipld.NavigableNode) error {
			node := ipld.ExtractIPLDNode(visitedNode)

			if len(node.Links()) > 0 {
				// Internal node, should be a `mdag.ProtoNode` containing a
				// `unixfs.FSNode` (see the `balanced` package for more details).
				fsNode, err := unixfs.ExtractFSNode(node)
				if err != nil {
					return err
				}

				// If there aren't enough size hints don't seek
				// (see the `io.EOF` handling error comment below).
				if fsNode.NumChildren() != len(node.Links()) {
					return ErrSeekNotSupported
				}

				// Internal nodes have no data, so just iterate through the
				// sizes of its children (advancing the child index of the
				// `dagWalker`) to find where we need to go down to next in
				// the search.
				for {
					childSize := fsNode.BlockSize(int(dr.dagWalker.ActiveChildIndex()))

					if childSize > uint64(left) {
						// This child's data contains the position requested
						// in `offset`, go down this child.
						return nil
					}

					// Else, skip this child.
					left -= int64(childSize)
					err := dr.dagWalker.NextChild()
					if err == ipld.ErrNextNoChild {
						// No more child nodes available, nothing to do,
						// the `Seek` will stop on its own.
						return nil
					} else if err != nil {
						return err
						// Pass along any other errors (that may in future
						// implementations be returned by `Next`) to stop
						// the search.
					}
				}

			} else {
				// Leaf node, seek inside its data.
				err := dr.saveNodeData(node)
				if err != nil {
					return err
				}

				_, err = dr.currentNodeData.Seek(left, io.SeekStart)
				if err != nil {
					return err
				}
				// The corner case of a DAG consisting only of a single (leaf)
				// node should make no difference here. In that case, where the
				// node doesn't have a parent UnixFS node with size hints, this
				// implementation would allow this `Seek` to be called with an
				// argument larger than the buffer size which normally wouldn't
				// happen (because we would skip the node based on the size
				// hint) but that would just mean that a future `CtxReadFull`
				// call would read no data from the `currentNodeData` buffer.
				// TODO: Re-check this reasoning.

				return nil
				// In the leaf node case the search will stop here.
			}
		})

		if err != nil {
			return 0, err
		}

		dr.offset = offset
		return dr.offset, nil

	case io.SeekCurrent:
		if offset == 0 {
			return dr.offset, nil
		}

		return dr.Seek(dr.offset+offset, io.SeekStart)
		// TODO: Performance. This can be improved supporting relative
		// searches in the `Walker` (see `Walker.Seek`).

	case io.SeekEnd:
		return dr.Seek(int64(dr.Size())+offset, io.SeekStart)

	default:
		return 0, errors.New("invalid whence")
	}
}

// Reset the reader position by resetting the `dagWalker` and discarding
// any partially used node's data in the `currentNodeData` buffer, used
// in the `SeekStart` case.
func (dr *dagReader) resetPosition() {
	dr.currentNodeData = nil
	dr.offset = 0

	dr.dagWalker = ipld.NewWalker(dr.ctx, ipld.NewNavigableIPLDNode(dr.rootNode, dr.serv))
	// TODO: This could be avoided (along with storing the `dr.rootNode` and
	// `dr.serv` just for this call) if `Reset` is supported in the `Walker`.
}
