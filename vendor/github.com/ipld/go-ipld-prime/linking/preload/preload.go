package preload

import (
	"context"

	"github.com/ipld/go-ipld-prime/datamodel"
)

// Loader is a function that will be called with a link discovered in a preload
// pass of a traversal. A preload pass can be used to collect all links in each
// block prior to traversal of that block, allowing for parallel (background)
// loading of blocks in anticipation of eventual actual load during traversal.
type Loader func(PreloadContext, Link)

// PreloadContext carries information about the current state of a traversal
// where a set of links that may be preloaded were encountered.
type PreloadContext struct {
	// Ctx is the familiar golang Context pattern.
	// Use this for cancellation, or attaching additional info
	// (for example, perhaps to pass auth tokens through to the storage functions).
	Ctx context.Context

	// Path where the link was encountered.  May be zero.
	//
	// Functions in the traversal package will set this automatically.
	BasePath datamodel.Path

	// Parent of the LinkNode.  May be zero.
	//
	// Functions in the traversal package will set this automatically.
	ParentNode datamodel.Node
}

// Link provides the link encountered during a preload pass, the node it was
// encountered on, and the segment of the path that led to the link.
type Link struct {
	Segment  datamodel.PathSegment
	LinkNode datamodel.Node
	Link     datamodel.Link
}
