package mfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	dag "github.com/ipfs/go-merkledag"
	ft "github.com/ipfs/go-unixfs"
	uio "github.com/ipfs/go-unixfs/io"

	cid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

var ErrNotYetImplemented = errors.New("not yet implemented")
var ErrInvalidChild = errors.New("invalid child node")
var ErrDirExists = errors.New("directory already has entry by that name")

// TODO: There's too much functionality associated with this structure,
// let's organize it (and if possible extract part of it elsewhere)
// and document the main features of `Directory` here.
type Directory struct {
	inode

	// Internal cache with added entries to the directory, its cotents
	// are synched with the underlying `unixfsDir` node in `sync()`.
	entriesCache map[string]FSNode

	lock sync.Mutex
	// TODO: What content is being protected here exactly? The entire directory?

	ctx context.Context

	// UnixFS directory implementation used for creating,
	// reading and editing directories.
	unixfsDir uio.Directory

	modTime time.Time
}

// NewDirectory constructs a new MFS directory.
//
// You probably don't want to call this directly. Instead, construct a new root
// using NewRoot.
func NewDirectory(ctx context.Context, name string, node ipld.Node, parent parent, dserv ipld.DAGService) (*Directory, error) {
	db, err := uio.NewDirectoryFromNode(dserv, node)
	if err != nil {
		return nil, err
	}

	return &Directory{
		inode: inode{
			name:       name,
			parent:     parent,
			dagService: dserv,
		},
		ctx:          ctx,
		unixfsDir:    db,
		entriesCache: make(map[string]FSNode),
		modTime:      time.Now(),
	}, nil
}

// GetCidBuilder gets the CID builder of the root node
func (d *Directory) GetCidBuilder() cid.Builder {
	return d.unixfsDir.GetCidBuilder()
}

// SetCidBuilder sets the CID builder
func (d *Directory) SetCidBuilder(b cid.Builder) {
	d.unixfsDir.SetCidBuilder(b)
}

// This method implements the `parent` interface. It first does the local
// update of the child entry in the underlying UnixFS directory and saves
// the newly created directory node with the updated entry in the DAG
// service. Then it propagates the update upwards (through this same
// interface) repeating the whole process in the parent.
func (d *Directory) updateChildEntry(c child) error {
	newDirNode, err := d.localUpdate(c)
	if err != nil {
		return err
	}

	// Continue to propagate the update process upwards
	// (all the way up to the root).
	return d.parent.updateChildEntry(child{d.name, newDirNode})
}

// This method implements the part of `updateChildEntry` that needs
// to be locked around: in charge of updating the UnixFS layer and
// generating the new node reflecting the update. It also stores the
// new node in the DAG layer.
func (d *Directory) localUpdate(c child) (*dag.ProtoNode, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	err := d.updateChild(c)
	if err != nil {
		return nil, err
	}
	// TODO: Clearly define how are we propagating changes to lower layers
	// like UnixFS.

	nd, err := d.unixfsDir.GetNode()
	if err != nil {
		return nil, err
	}

	pbnd, ok := nd.(*dag.ProtoNode)
	if !ok {
		return nil, dag.ErrNotProtobuf
	}

	err = d.dagService.Add(d.ctx, nd)
	if err != nil {
		return nil, err
	}

	return pbnd.Copy().(*dag.ProtoNode), nil
	// TODO: Why do we need a copy?
}

// Update child entry in the underlying UnixFS directory.
func (d *Directory) updateChild(c child) error {
	err := d.unixfsDir.AddChild(d.ctx, c.Name, c.Node)
	if err != nil {
		return err
	}

	d.modTime = time.Now()

	return nil
}

func (d *Directory) Type() NodeType {
	return TDir
}

// childNode returns a FSNode under this directory by the given name if it exists.
// it does *not* check the cached dirs and files
func (d *Directory) childNode(name string) (FSNode, error) {
	nd, err := d.childFromDag(name)
	if err != nil {
		return nil, err
	}

	return d.cacheNode(name, nd)
}

// cacheNode caches a node into d.childDirs or d.files and returns the FSNode.
func (d *Directory) cacheNode(name string, nd ipld.Node) (FSNode, error) {
	switch nd := nd.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(nd.Data())
		if err != nil {
			return nil, err
		}

		switch fsn.Type() {
		case ft.TDirectory, ft.THAMTShard:
			ndir, err := NewDirectory(d.ctx, name, nd, d, d.dagService)
			if err != nil {
				return nil, err
			}

			d.entriesCache[name] = ndir
			return ndir, nil
		case ft.TFile, ft.TRaw, ft.TSymlink:
			nfi, err := NewFile(name, nd, d, d.dagService)
			if err != nil {
				return nil, err
			}
			d.entriesCache[name] = nfi
			return nfi, nil
		case ft.TMetadata:
			return nil, ErrNotYetImplemented
		default:
			return nil, ErrInvalidChild
		}
	case *dag.RawNode:
		nfi, err := NewFile(name, nd, d, d.dagService)
		if err != nil {
			return nil, err
		}
		d.entriesCache[name] = nfi
		return nfi, nil
	default:
		return nil, fmt.Errorf("unrecognized node type in cache node")
	}
}

// Child returns the child of this directory by the given name
func (d *Directory) Child(name string) (FSNode, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.childUnsync(name)
}

func (d *Directory) Uncache(name string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.entriesCache, name)
}

// childFromDag searches through this directories dag node for a child link
// with the given name
func (d *Directory) childFromDag(name string) (ipld.Node, error) {
	return d.unixfsDir.Find(d.ctx, name)
}

// childUnsync returns the child under this directory by the given name
// without locking, useful for operations which already hold a lock
func (d *Directory) childUnsync(name string) (FSNode, error) {
	entry, ok := d.entriesCache[name]
	if ok {
		return entry, nil
	}

	return d.childNode(name)
}

type NodeListing struct {
	Name string
	Type int
	Size int64
	Hash string
}

func (d *Directory) ListNames(ctx context.Context) ([]string, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	var out []string
	err := d.unixfsDir.ForEachLink(ctx, func(l *ipld.Link) error {
		out = append(out, l.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (d *Directory) List(ctx context.Context) ([]NodeListing, error) {
	var out []NodeListing
	err := d.ForEachEntry(ctx, func(nl NodeListing) error {
		out = append(out, nl)
		return nil
	})
	return out, err
}

func (d *Directory) ForEachEntry(ctx context.Context, f func(NodeListing) error) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.unixfsDir.ForEachLink(ctx, func(l *ipld.Link) error {
		c, err := d.childUnsync(l.Name)
		if err != nil {
			return err
		}

		nd, err := c.GetNode()
		if err != nil {
			return err
		}

		child := NodeListing{
			Name: l.Name,
			Type: int(c.Type()),
			Hash: nd.Cid().String(),
		}

		if c, ok := c.(*File); ok {
			size, err := c.Size()
			if err != nil {
				return err
			}
			child.Size = size
		}

		return f(child)
	})
}

func (d *Directory) Mkdir(name string) (*Directory, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	fsn, err := d.childUnsync(name)
	if err == nil {
		switch fsn := fsn.(type) {
		case *Directory:
			return fsn, os.ErrExist
		case *File:
			return nil, os.ErrExist
		default:
			return nil, fmt.Errorf("unrecognized type: %#v", fsn)
		}
	}

	ndir := ft.EmptyDirNode()
	ndir.SetCidBuilder(d.GetCidBuilder())

	err = d.dagService.Add(d.ctx, ndir)
	if err != nil {
		return nil, err
	}

	err = d.unixfsDir.AddChild(d.ctx, name, ndir)
	if err != nil {
		return nil, err
	}

	dirobj, err := NewDirectory(d.ctx, name, ndir, d, d.dagService)
	if err != nil {
		return nil, err
	}

	d.entriesCache[name] = dirobj
	return dirobj, nil
}

func (d *Directory) Unlink(name string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	delete(d.entriesCache, name)

	return d.unixfsDir.RemoveChild(d.ctx, name)
}

func (d *Directory) Flush() error {
	nd, err := d.GetNode()
	if err != nil {
		return err
	}

	return d.parent.updateChildEntry(child{d.name, nd})
}

// AddChild adds the node 'nd' under this directory giving it the name 'name'
func (d *Directory) AddChild(name string, nd ipld.Node) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	_, err := d.childUnsync(name)
	if err == nil {
		return ErrDirExists
	}

	err = d.dagService.Add(d.ctx, nd)
	if err != nil {
		return err
	}

	err = d.unixfsDir.AddChild(d.ctx, name, nd)
	if err != nil {
		return err
	}

	d.modTime = time.Now()
	return nil
}

func (d *Directory) sync() error {
	for name, entry := range d.entriesCache {
		nd, err := entry.GetNode()
		if err != nil {
			return err
		}

		err = d.updateChild(child{name, nd})
		if err != nil {
			return err
		}
	}

	// TODO: Should we clean the cache here?

	return nil
}

func (d *Directory) Path() string {
	cur := d
	var out string
	for cur != nil {
		switch parent := cur.parent.(type) {
		case *Directory:
			out = path.Join(cur.name, out)
			cur = parent
		case *Root:
			return "/" + out
		default:
			panic("directory parent neither a directory nor a root")
		}
	}
	return out
}

func (d *Directory) GetNode() (ipld.Node, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	err := d.sync()
	if err != nil {
		return nil, err
	}

	nd, err := d.unixfsDir.GetNode()
	if err != nil {
		return nil, err
	}

	err = d.dagService.Add(d.ctx, nd)
	if err != nil {
		return nil, err
	}

	return nd.Copy(), err
}
