package traversal

import (
	"context"
	"fmt"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	"github.com/ipld/go-ipld-prime/linking/preload"
)

// This file defines interfaces for things users provide,
//  plus a few of the parameters they'll need to receieve.
//--------------------------------------------------------

// VisitFn is a read-only visitor.
type VisitFn func(Progress, datamodel.Node) error

// TransformFn is like a visitor that can also return a new Node to replace the visited one.
type TransformFn func(Progress, datamodel.Node) (datamodel.Node, error)

// AdvVisitFn is like VisitFn, but for use with AdvTraversal: it gets additional arguments describing *why* this node is visited.
type AdvVisitFn func(Progress, datamodel.Node, VisitReason) error

// VisitReason provides additional information to traversals using AdvVisitFn.
type VisitReason byte

const (
	// VisitReason_SelectionMatch tells AdvVisitFn that this node was explicitly selected.  (This is the set of nodes that VisitFn is called for.)
	VisitReason_SelectionMatch VisitReason = 'm'
	// VisitReason_SelectionParent tells AdvVisitFn that this node is a parent of one that will be explicitly selected.  (These calls only happen if the feature is enabled -- enabling parent detection requires a different algorithm and adds some overhead.)
	VisitReason_SelectionParent VisitReason = 'p'
	// VisitReason_SelectionCandidate tells AdvVisitFn that this node was visited while searching for selection matches.  It is not necessarily implied that any explicit match will be a child of this node; only that we had to consider it.  (Merkle-proofs generally need to include any node in this group.)
	VisitReason_SelectionCandidate VisitReason = 'x'
)

// Progress tracks a traversal as it proceeds. It is used initially to begin a traversal, and it is then passed to the visit function as the traversal proceeds.
//
// As the traversal descends into the graph, new Progress values are created and passed to the visit function with updated properties representing the current state of the traversal.
//
// Most customization of a traversal is done by setting a Cfg property on a Progress before beginning the traversal.
// Typical customization involves setting a LinkSystem for link loading and/or tracking.
//
// Advanced traversal control options, such as LinkVisitOnlyOnce and StartAtPath, are also available in the Cfg but may have surprising effects on traversal behavior; be careful when using them.
//
// Budgets are set on the Progress option because a Budget, while set at the beginning of a traversal, is also updated as the traversal proceeds, with its fields being monotonically decremented.
// Beware of using Budgets in tandem with a Preloader! The preloader discovers links in a lateral scan of a whole block, before rewinding for a depth-first walk for traversal-proper.
// Budgets are intended to be used for the depth-first walk, and there is no way to know ahead of time how the budget may impact the lateral parts of the graph that the preloader encounters.
// Currently a best-guess approach is used to try and have the preloader adhere to the budget, but with typical real-world graphs, this is likely to be inaccurate.
// In the case of inaccuracies, the budget will be properly applied to the traversal-proper, but the preloader may receive a different set of links than the traversal-proper will.
type Progress struct {
	// Cfg is the configuration for the traversal, set by user.
	Cfg *Config

	// Budget, if present, tracks "budgets" for how many more steps we're willing to take before we should halt.
	// Budget is initially set by user, but is then updated as the traversal proceeds.
	Budget *Budget

	// Path is how we reached the current point in the traversal.
	Path datamodel.Path

	// LastBlock stores the Path and Link of the last block edge we had to load.  (It will always be zero in traversals with no linkloader.)
	LastBlock struct {
		Path datamodel.Path
		Link datamodel.Link
	}

	// PastStartAtPath indicates whether the traversal has progressed passed the StartAtPath in the config -- use to avoid path checks when inside a sub portion of a DAG that is entirely inside the "not-skipped" portion of a traversal
	PastStartAtPath bool

	// SeenLinks is a set used to remember which links have been visited before, if Cfg.LinkVisitOnlyOnce is true.
	SeenLinks map[datamodel.Link]struct{}
}

// Config is a set of options for a traversal. Set a Config on a Progress to customize the traversal.
type Config struct {
	// Ctx is the context carried through a traversal.
	// Optional; use it if you need cancellation.
	Ctx context.Context

	// LinkSystem is used for automatic link loading, and also any storing if mutation features (e.g. traversal.Transform) are used.
	LinkSystem linking.LinkSystem

	// LinkTargetNodePrototypeChooser is a chooser for Node implementations to produce during automatic link traversal.
	LinkTargetNodePrototypeChooser LinkTargetNodePrototypeChooser

	// LinkVisitOnlyOnce controls repeat-link visitation.
	// By default, we visit across links wherever we see them again, even if we've visited them before, because the reason for visiting might be different than it was before since we got to it via a different path.
	// If set to true, track links we've seen before in Progress.SeenLinks and do not visit them again.
	// Note that sufficiently complex selectors may require valid revisiting of some links, so setting this to true can change behavior noticably and should be done with care.
	LinkVisitOnlyOnce bool

	// StartAtPath, if set, causes a traversal to skip forward until passing this path, and only then begins calling visit functions.
	// Block loads will also be skipped wherever possible.
	StartAtPath datamodel.Path

	// Preloader receives links within each block prior to traversal-proper by performing a lateral scan of a block without descending into links themselves before backing up and doing a traversal-proper.
	// This can be used to asynchronously load blocks that will be required at a later phase of the retrieval, or even to load blocks in a different order than the traversal would otherwise do.
	// Preload calls are not de-duplicated, it is up to the receiver to do so if desired.
	// Beware of using both Budget and Preloader!  See the documentation on Progress for more information on this usage and the likely surprising effects.
	Preloader preload.Loader
}

// Budget is a set of monotonically-decrementing "budgets" for how many more steps we're willing to take before we should halt.
//
// The fields of Budget are described as "monotonically-decrementing", because that's what the traversal library will do with them,
// but they are user-accessable and can be reset to higher numbers again by code in the visitor callbacks.  This is not recommended (why?), but possible.

// If you set any budgets (by having a non-nil Progress.Budget field), you must set some value for all of them.
// Traversal halts when _any_ of the budgets reaches zero.
// The max value of an int (math.MaxInt64) is acceptable for any budget you don't care about.
//
// Beware of using both Budget and Preloader!  See the documentation on Progress for more information on this usage and the likely surprising effects.
type Budget struct {
	// NodeBudget is a monotonically-decrementing "budget" for how many more nodes we're willing to visit before halting.
	NodeBudget int64
	// LinkBudget is a monotonically-decrementing "budget" for how many more links we're willing to load before halting.
	// (This is not aware of any caching; it's purely in terms of links encountered and traversed.)
	LinkBudget int64
}

// Clone returns a copy of the budget.
func (b *Budget) Clone() *Budget {
	if b == nil {
		return nil
	}
	return &Budget{
		NodeBudget: b.NodeBudget,
		LinkBudget: b.LinkBudget,
	}
}

// LinkTargetNodePrototypeChooser is a function that returns a NodePrototype based on
// the information in a Link and/or its LinkContext.
//
// A LinkTargetNodePrototypeChooser can be used in a traversal.Config to be clear about
// what kind of Node implementation to use when loading a Link.
// In a simple example, it could constantly return a `basicnode.Prototype.Any`.
// In a more complex example, a program using `bind` over native Go types
// could decide what kind of native type is expected, and return a
// `bind.NodeBuilder` for that specific concrete native type.
type LinkTargetNodePrototypeChooser func(datamodel.Link, linking.LinkContext) (datamodel.NodePrototype, error)

// SkipMe is a signalling "error" which can be used to tell traverse to skip some data.
//
// SkipMe can be returned by the Config.LinkLoader to skip entire blocks without aborting the walk.
// (This can be useful if you know you don't have data on hand,
// but want to continue the walk in other areas anyway;
// or, if you're doing a way where you know that it's valid to memoize seen
// areas based on Link alone.)
type SkipMe struct{}

func (SkipMe) Error() string {
	return "skip"
}

type ErrBudgetExceeded struct {
	BudgetKind string // "node"|"link"
	Path       datamodel.Path
	Link       datamodel.Link // only present if BudgetKind=="link"
}

func (e *ErrBudgetExceeded) Error() string {
	msg := fmt.Sprintf("traversal budget exceeded: budget for %ss reached zero while on path %q", e.BudgetKind, e.Path)
	if e.Link != nil {
		msg += fmt.Sprintf(" (link: %q)", e.Link)
	}
	return msg
}

func (e *ErrBudgetExceeded) Is(target error) bool {
	_, ok := target.(*ErrBudgetExceeded)
	return ok
}
