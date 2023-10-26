package unixfile

import (
	"context"
	"errors"

	ft "github.com/ipfs/go-unixfs"
	uio "github.com/ipfs/go-unixfs/io"

	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
)

// Number to file to prefetch in directories
// TODO: should we allow setting this via context hint?
const prefetchFiles = 4

type ufsDirectory struct {
	ctx   context.Context
	dserv ipld.DAGService
	dir   uio.Directory
	size  int64
}

type ufsIterator struct {
	ctx   context.Context
	files chan *ipld.Link
	dserv ipld.DAGService

	curName string
	curFile files.Node

	err   error
	errCh chan error
}

func (it *ufsIterator) Name() string {
	return it.curName
}

func (it *ufsIterator) Node() files.Node {
	return it.curFile
}

func (it *ufsIterator) Next() bool {
	if it.err != nil {
		return false
	}

	var l *ipld.Link
	var ok bool
	for !ok {
		if it.files == nil && it.errCh == nil {
			return false
		}
		select {
		case l, ok = <-it.files:
			if !ok {
				it.files = nil
			}
		case err := <-it.errCh:
			it.errCh = nil
			it.err = err

			if err != nil {
				return false
			}
		}
	}

	it.curFile = nil

	nd, err := l.GetNode(it.ctx, it.dserv)
	if err != nil {
		it.err = err
		return false
	}

	it.curName = l.Name
	it.curFile, it.err = NewUnixfsFile(it.ctx, it.dserv, nd)
	return it.err == nil
}

func (it *ufsIterator) Err() error {
	return it.err
}

func (d *ufsDirectory) Close() error {
	return nil
}

func (d *ufsDirectory) Entries() files.DirIterator {
	fileCh := make(chan *ipld.Link, prefetchFiles)
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.dir.ForEachLink(d.ctx, func(link *ipld.Link) error {
			if d.ctx.Err() != nil {
				return d.ctx.Err()
			}
			select {
			case fileCh <- link:
			case <-d.ctx.Done():
				return d.ctx.Err()
			}
			return nil
		})

		close(errCh)
		close(fileCh)
	}()

	return &ufsIterator{
		ctx:   d.ctx,
		files: fileCh,
		errCh: errCh,
		dserv: d.dserv,
	}
}

func (d *ufsDirectory) Size() (int64, error) {
	return d.size, nil
}

type ufsFile struct {
	uio.DagReader
}

func (f *ufsFile) Size() (int64, error) {
	return int64(f.DagReader.Size()), nil
}

func newUnixfsDir(ctx context.Context, dserv ipld.DAGService, nd *dag.ProtoNode) (files.Directory, error) {
	dir, err := uio.NewDirectoryFromNode(dserv, nd)
	if err != nil {
		return nil, err
	}

	size, err := nd.Size()
	if err != nil {
		return nil, err
	}

	return &ufsDirectory{
		ctx:   ctx,
		dserv: dserv,

		dir:  dir,
		size: int64(size),
	}, nil
}

func NewUnixfsFile(ctx context.Context, dserv ipld.DAGService, nd ipld.Node) (files.Node, error) {
	switch dn := nd.(type) {
	case *dag.ProtoNode:
		fsn, err := ft.FSNodeFromBytes(dn.Data())
		if err != nil {
			return nil, err
		}
		if fsn.IsDir() {
			return newUnixfsDir(ctx, dserv, dn)
		}
		if fsn.Type() == ft.TSymlink {
			return files.NewLinkFile(string(fsn.Data()), nil), nil
		}

	case *dag.RawNode:
	default:
		return nil, errors.New("unknown node type")
	}

	dr, err := uio.NewDagReader(ctx, nd, dserv)
	if err != nil {
		return nil, err
	}

	return &ufsFile{
		DagReader: dr,
	}, nil
}

var _ files.Directory = &ufsDirectory{}
var _ files.File = &ufsFile{}
