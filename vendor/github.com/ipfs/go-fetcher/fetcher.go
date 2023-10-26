package fetcher

import (
	"context"

	"github.com/ipld/go-ipld-prime"
)

// Fetcher is an interface for reading from a dag. Reads may be local or remote, and may employ data exchange
// protocols like graphsync and bitswap
type Fetcher interface {
	// NodeMatching traverses a node graph starting with the provided root node using the given selector node and
	// possibly crossing block boundaries. Each matched node is passed as FetchResult to the callback. Errors returned
	// from callback will halt the traversal. The sequence of events is: NodeMatching begins, the callback is called zero
	// or more times with a FetchResult, then NodeMatching returns.
	NodeMatching(ctx context.Context, root ipld.Node, selector ipld.Node, cb FetchCallback) error

	// BlockOfType fetches a node graph of the provided type corresponding to single block by link.
	BlockOfType(ctx context.Context, link ipld.Link, nodePrototype ipld.NodePrototype) (ipld.Node, error)

	// BlockMatchingOfType traverses a node graph starting with the given root link using the given selector node and
	// possibly crossing block boundaries. The nodes will be typed using the provided prototype. Each matched node is
	// passed as a FetchResult to the callback. Errors returned from callback will halt the traversal.
	// The sequence of events is: BlockMatchingOfType begins, the callback is called zero or more times with a
	// FetchResult, then BlockMatchingOfType returns.
	BlockMatchingOfType(
		ctx context.Context,
		root ipld.Link,
		selector ipld.Node,
		nodePrototype ipld.NodePrototype,
		cb FetchCallback) error

	// Uses the given link to pick a prototype to build the linked node.
	PrototypeFromLink(link ipld.Link) (ipld.NodePrototype, error)
}

// FetchResult is a single node read as part of a dag operation called on a fetcher
type FetchResult struct {
	Node          ipld.Node
	Path          ipld.Path
	LastBlockPath ipld.Path
	LastBlockLink ipld.Link
}

// FetchCallback is called for each node traversed during a fetch
type FetchCallback func(result FetchResult) error

// Factory is anything that can create new sessions of the fetcher
type Factory interface {
	NewSession(ctx context.Context) Fetcher
}
