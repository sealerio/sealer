package mfs

import (
	"context"
	"fmt"
	"os"
	gopath "path"
	"strings"

	path "github.com/ipfs/go-path"

	cid "github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

// TODO: Evaluate moving all this operations to as `Root`
// methods, since all of them use it as its first argument
// and there is no clear documentation that explains this
// separation.

// Mv moves the file or directory at 'src' to 'dst'
// TODO: Document what the strings 'src' and 'dst' represent.
func Mv(r *Root, src, dst string) error {
	srcDirName, srcFname := gopath.Split(src)

	var dstDirName string
	var dstFname string
	if dst[len(dst)-1] == '/' {
		dstDirName = dst
		dstFname = srcFname
	} else {
		dstDirName, dstFname = gopath.Split(dst)
	}

	// get parent directories of both src and dest first
	dstDir, err := lookupDir(r, dstDirName)
	if err != nil {
		return err
	}

	srcDir, err := lookupDir(r, srcDirName)
	if err != nil {
		return err
	}

	srcObj, err := srcDir.Child(srcFname)
	if err != nil {
		return err
	}

	nd, err := srcObj.GetNode()
	if err != nil {
		return err
	}

	fsn, err := dstDir.Child(dstFname)
	if err == nil {
		switch n := fsn.(type) {
		case *File:
			_ = dstDir.Unlink(dstFname)
		case *Directory:
			dstDir = n
			dstFname = srcFname
		default:
			return fmt.Errorf("unexpected type at path: %s", dst)
		}
	} else if err != os.ErrNotExist {
		return err
	}

	err = dstDir.AddChild(dstFname, nd)
	if err != nil {
		return err
	}

	if srcDir.name == dstDir.name && srcFname == dstFname {
		return nil
	}

	return srcDir.Unlink(srcFname)
}

func lookupDir(r *Root, path string) (*Directory, error) {
	di, err := Lookup(r, path)
	if err != nil {
		return nil, err
	}

	d, ok := di.(*Directory)
	if !ok {
		return nil, fmt.Errorf("%s is not a directory", path)
	}

	return d, nil
}

// PutNode inserts 'nd' at 'path' in the given mfs
// TODO: Rename or clearly document that this is not about nodes but actually
// MFS files/directories (that in the underlying representation can be
// considered as just nodes).
// TODO: Document why are we handling IPLD nodes in the first place when we
// are actually referring to files/directories (that is, it can't be any
// node, it has to have a specific format).
// TODO: Can this function add directories or just files? What would be the
// difference between adding a directory with this method and creating it
// with `Mkdir`.
func PutNode(r *Root, path string, nd ipld.Node) error {
	dirp, filename := gopath.Split(path)
	if filename == "" {
		return fmt.Errorf("cannot create file with empty name")
	}

	pdir, err := lookupDir(r, dirp)
	if err != nil {
		return err
	}

	return pdir.AddChild(filename, nd)
}

// MkdirOpts is used by Mkdir
type MkdirOpts struct {
	Mkparents  bool
	Flush      bool
	CidBuilder cid.Builder
}

// Mkdir creates a directory at 'path' under the directory 'd', creating
// intermediary directories as needed if 'mkparents' is set to true
func Mkdir(r *Root, pth string, opts MkdirOpts) error {
	if pth == "" {
		return fmt.Errorf("no path given to Mkdir")
	}
	parts := path.SplitList(pth)
	if parts[0] == "" {
		parts = parts[1:]
	}

	// allow 'mkdir /a/b/c/' to create c
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 0 {
		// this will only happen on 'mkdir /'
		if opts.Mkparents {
			return nil
		}
		return fmt.Errorf("cannot create directory '/': Already exists")
	}

	cur := r.GetDirectory()
	for i, d := range parts[:len(parts)-1] {
		fsn, err := cur.Child(d)
		if err == os.ErrNotExist && opts.Mkparents {
			mkd, err := cur.Mkdir(d)
			if err != nil {
				return err
			}
			if opts.CidBuilder != nil {
				mkd.SetCidBuilder(opts.CidBuilder)
			}
			fsn = mkd
		} else if err != nil {
			return err
		}

		next, ok := fsn.(*Directory)
		if !ok {
			return fmt.Errorf("%s was not a directory", path.Join(parts[:i]))
		}
		cur = next
	}

	final, err := cur.Mkdir(parts[len(parts)-1])
	if err != nil {
		if !opts.Mkparents || err != os.ErrExist || final == nil {
			return err
		}
	}
	if opts.CidBuilder != nil {
		final.SetCidBuilder(opts.CidBuilder)
	}

	if opts.Flush {
		err := final.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

// Lookup extracts the root directory and performs a lookup under it.
// TODO: Now that the root is always a directory, can this function
// be collapsed with `DirLookup`? Or at least be made a method of `Root`?
func Lookup(r *Root, path string) (FSNode, error) {
	dir := r.GetDirectory()

	return DirLookup(dir, path)
}

// DirLookup will look up a file or directory at the given path
// under the directory 'd'
func DirLookup(d *Directory, pth string) (FSNode, error) {
	pth = strings.Trim(pth, "/")
	parts := path.SplitList(pth)
	if len(parts) == 1 && parts[0] == "" {
		return d, nil
	}

	var cur FSNode
	cur = d
	for i, p := range parts {
		chdir, ok := cur.(*Directory)
		if !ok {
			return nil, fmt.Errorf("cannot access %s: Not a directory", path.Join(parts[:i+1]))
		}

		child, err := chdir.Child(p)
		if err != nil {
			return nil, err
		}

		cur = child
	}
	return cur, nil
}

// TODO: Document this function and link its functionality
// with the republisher.
func FlushPath(ctx context.Context, rt *Root, pth string) (ipld.Node, error) {
	nd, err := Lookup(rt, pth)
	if err != nil {
		return nil, err
	}

	err = nd.Flush()
	if err != nil {
		return nil, err
	}

	rt.repub.WaitPub(ctx)
	return nd.GetNode()
}
