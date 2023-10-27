package mfs

import (
	ipld "github.com/ipfs/go-ipld-format"
)

// inode abstracts the common characteristics of the MFS `File`
// and `Directory`. All of its attributes are initialized at
// creation.
type inode struct {
	// name of this `inode` in the MFS path (the same value
	// is also stored as the name of the DAG link).
	name string

	// parent directory of this `inode` (which may be the `Root`).
	parent parent

	// dagService used to store modifications made to the contents
	// of the file or directory the `inode` belongs to.
	dagService ipld.DAGService
}
