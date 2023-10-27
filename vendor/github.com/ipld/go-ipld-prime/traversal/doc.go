// Package traversal provides functional utilities for traversing and
// transforming IPLD graphs.
//
// Two primary types of traversal are implemented in this package: "Focus" and
// "Walk". Both types have a "Transforming" variant, which supports mutation
// through emulated copy-on-write tree rebuilding.
//
// Traversal operations use the Progress type for configuration and state
// tracking. Helper functions such as Focus and Walk exist to avoid manual setup
// of a Progress struct, but they cannot cross link boundaries without a
// LinkSystem, which needs to be configured on the Progress struct.
//
// A typical traversal operation involves creating a Progress struct, setting up
// the LinkSystem, and calling one of the Focus or Walk functions on the
// Progress object. Various other configuration options are available when
// traversing this way.
//
// # Focus
//
// "Focus" and "Get" functions provide syntactic sugar for using ipld.Path to
// access Nodes deep within a graph.
//
// "FocusedTransform" resembles "Focus" but supports user-defined mutation using
// its TransformFn.
//
// # Walk
//
// "Walk" functions perform a recursive walk of a Node graph, applying visitor
// functions to matched parts of the graph.
//
// The selector sub-package offers a declarative mechanism for guiding
// traversals and filtering relevant Nodes.
// (Refer to the selector sub-package for more details.)
//
// "WalkLocal" is a special case of Walk that doesn't require a selector. It
// walks a local graph, not crossing link boundaries, and calls its VisitFn for
// each encountered Node.
//
// "WalkMatching" traverses according to a selector, calling the VisitFn for
// each match based on the selector's matching rules.
//
// "WalkAdv" performs the same traversal as WalkMatching, but calls its
// AdvVisitFn on every Node, regardless of whether it matches the selector.
//
// "WalkTransforming" resembles "WalkMatching" but supports user-defined
// mutation using its TransformFn.
//
// # Usage Notes
//
// These functions work via callbacks, performing traversal and calling a
// user-provided function with a handle to the reached Node(s). Further "Focus"
// and "Walk" operations can be performed recursively within this callback if
// desired.
//
// All traversal functions operate on a Progress object, except "WalkLocal",
// which can be configured with a LinkSystem for automatic resolution and
// loading of new Node trees when IPLD Links are encountered.
//
// The "*Transform" methods are best suited for point-mutation patterns. For
// more general transformations, use the read-only systems (e.g., Focus,
// Traverse) and handle accumulation in the visitor functions.
//
// A common use case for walking traversal is running a selector over a graph
// and noting all the blocks it uses. This is achieved by configuring a
// LinkSystem that can handle and observe block loads. Be aware that a selector
// might visit the same block multiple times during a traversal, as IPLD graphs
// often form "diamond patterns" with the same block referenced from multiple
// locations.
//
// The LinkVisitOnlyOnce option can be used to avoid duplicate loads, but it
// must be used carefully with non-trivial selectors, where repeat visits of
// the same block may be essential for traversal or visit callbacks.
//
// A Budget can be set at the beginning of a traversal to limit the number of
// Nodes and/or Links encountered before failing the traversal (with the
// ErrBudgetExceeded error).
//
// The "Preloader" option provides a way to parallelize block loading in
// environments where block loading is a high-latency operation (such as
// fetching over the network).
// The traversal operation itself is not parallel and will proceed strictly
// according to path or selector order. However, a Preloader can be used to load
// blocks asynchronously, and prepare the LinkSystem that the traversal is using
// with already-loaded blocks.
//
// A Preloader and a Budget option can be used on the same traversal, BUT the
// Preloader may not receive the same links that the traversal wants to load
// from the LinkSystem. Use with care. See notes below.
package traversal

// Why only "point-mutation"?  This use-case gets core library support because
// it's both high utility and highly clear how to implement it.
// More advanced transformations are nontrivial to provide generalized support
// for, for three reasons: efficiency is hard; not all existing research into
// categorical recursion schemes is necessarily applicable without modification
// (efficient behavior in a merkle-tree context is not the same as efficient
// behavior on uniform memory!); and we have the further compounding complexity
// of the range of choices available for underlying Node implementation.
// Therefore, attempts at generalization are not included here; handling these
// issues in concrete cases is easy, so we call it an application logic concern.
// However, exploring categorical recursion schemes as a library is encouraged!)
