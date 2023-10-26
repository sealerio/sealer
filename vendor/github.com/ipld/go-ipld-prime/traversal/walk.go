package traversal

import (
	"errors"
	"fmt"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking"
	"github.com/ipld/go-ipld-prime/linking/preload"
	"github.com/ipld/go-ipld-prime/traversal/selector"
)

// phase is an internal enum used to track the current phase of a walk. It's
// used to control for a preload pass over a block if one is required.
type phase int

const (
	phasePreload  phase = iota
	phaseTraverse phase = iota
)

// WalkLocal walks a tree of Nodes, visiting each of them,
// and calling the given VisitFn on all of them;
// it does not traverse any links.
//
// WalkLocal can skip subtrees if the VisitFn returns SkipMe,
// but lacks any other options for controlling or directing the visit;
// consider using some of the various Walk functions with Selector parameters if you want more control.
func WalkLocal(n datamodel.Node, fn VisitFn) error {
	return Progress{}.WalkLocal(n, fn)
}

// WalkMatching walks a graph of Nodes, deciding which to visit by applying a Selector,
// and calling the given VisitFn on those that the Selector deems a match.
//
// This function is a helper function which starts a new walk with default configuration.
// It cannot cross links automatically (since this requires configuration).
// Use the equivalent WalkMatching function on the Progress structure
// for more advanced and configurable walks.
func WalkMatching(n datamodel.Node, s selector.Selector, fn VisitFn) error {
	return Progress{}.WalkMatching(n, s, fn)
}

// WalkAdv is identical to WalkMatching, except it is called for *all* nodes
// visited (not just matching nodes), together with the reason for the visit.
// An AdvVisitFn is used instead of a VisitFn, so that the reason can be provided.
//
// This function is a helper function which starts a new walk with default configuration.
// It cannot cross links automatically (since this requires configuration).
// Use the equivalent WalkAdv function on the Progress structure
// for more advanced and configurable walks.
func WalkAdv(n datamodel.Node, s selector.Selector, fn AdvVisitFn) error {
	return Progress{}.WalkAdv(n, s, fn)
}

// WalkTransforming walks a graph of Nodes, deciding which to alter by applying a Selector,
// and calls the given TransformFn to decide what new node to replace the visited node with.
// A new Node tree will be returned (the original is unchanged).
//
// This function is a helper function which starts a new walk with default configuration.
// It cannot cross links automatically (since this requires configuration).
// Use the equivalent WalkTransforming function on the Progress structure
// for more advanced and configurable walks.
func WalkTransforming(n datamodel.Node, s selector.Selector, fn TransformFn) (datamodel.Node, error) {
	return Progress{}.WalkTransforming(n, s, fn)
}

// WalkMatching walks a graph of Nodes, deciding which to visit by applying a Selector,
// and calling the given VisitFn on those that the Selector deems a match.
//
// WalkMatching is a read-only traversal.
// See WalkTransforming if looking for a way to do "updates" to a tree of nodes.
//
// Provide configuration to this process using the Config field in the Progress object.
//
// This walk will automatically cross links, but requires some configuration
// with link loading functions to do so.
//
// Traversals are defined as visiting a (node,path) tuple.
// This is important to note because when walking DAGs with Links,
// it means you may visit the same node multiple times
// due to having reached it via a different path.
// (You can prevent this by using a LinkLoader function which memoizes a set of
// already-visited Links, and returns a SkipMe when encountering them again.)
//
// WalkMatching (and the other traversal functions) can be used again again inside the VisitFn!
// By using the traversal.Progress handed to the VisitFn,
// the Path recorded of the traversal so far will continue to be extended,
// and thus continued nested uses of Walk and Focus will see the fully contextualized Path.
//
// WalkMatching can be configured to run with a Preloader.
// When a Preloader is configured, the walk will first do a "preload" pass over the initial,
// root tree up to link boundaries and report any links encountered to the preloader.
// It will then perform a second pass over the tree, calling the VisitFn where necessary as per normal WalkMatching behavior.
// This two-pass operation will continue for each block loaded, allowing the preloader to
// potentially asynchronously preload any blocks that are going to be encountered at a future point in the walk.
func (prog Progress) WalkMatching(n datamodel.Node, s selector.Selector, fn VisitFn) error {
	prog.init()
	return prog.walkBlock(n, s, func(prog Progress, n datamodel.Node, tr VisitReason) error {
		if tr != VisitReason_SelectionMatch {
			return nil
		}
		return fn(prog, n)
	})
}

// WalkLocal is the same as the package-scope function of the same name,
// but considers an existing Progress state (and any config it might reference).
func (prog Progress) WalkLocal(n datamodel.Node, fn VisitFn) error {
	if err := prog.checkNodeBudget(); err != nil {
		return err
	}

	// Visit the current node.
	if err := fn(prog, n); err != nil {
		if _, ok := err.(SkipMe); ok {
			return nil
		}
		return err
	}
	// Recurse on nodes with a recursive kind; otherwise just return.
	switch n.Kind() {
	case datamodel.Kind_Map:
		for itr := n.MapIterator(); !itr.Done(); {
			k, v, err := itr.Next()
			if err != nil {
				return err
			}
			ks, _ := k.AsString()
			progNext := prog
			progNext.Path = prog.Path.AppendSegmentString(ks)
			if err := progNext.WalkLocal(v, fn); err != nil {
				return err
			}
		}
		return nil
	case datamodel.Kind_List:
		for itr := n.ListIterator(); !itr.Done(); {
			idx, v, err := itr.Next()
			if err != nil {
				return err
			}
			progNext := prog
			progNext.Path = prog.Path.AppendSegmentInt(idx)
			if err := progNext.WalkLocal(v, fn); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

// WalkAdv is identical to WalkMatching, except it is called for *all* nodes
// visited (not just matching nodes), together with the reason for the visit.
// An AdvVisitFn is used instead of a VisitFn, so that the reason can be provided.
func (prog Progress) WalkAdv(n datamodel.Node, s selector.Selector, fn AdvVisitFn) error {
	prog.init()
	return prog.walkBlock(n, s, fn)
}

// walkBlock anchors a walk at the beginning of the traversal and at the
// beginning of each new link traversed. This allows us to do a preload phase if
// we have a preloader configured.
func (prog Progress) walkBlock(n datamodel.Node, s selector.Selector, visitFn AdvVisitFn) error {
	ph := phaseTraverse
	var budget *Budget

	if prog.Cfg.Preloader != nil {
		ph = phasePreload
		// preserve the budget so we can reset it for the second pass; it will
		// likely not correctly apply during the preload phase because it
		// doesn't descend into links first. But we'll use it anyway as a
		// best-guess because we have nothing better
		budget = prog.Budget.Clone()
	}

	// First pass.
	err := prog.walkAdv(ph, n, s, visitFn)
	if err != nil && (ph != phasePreload || !errors.Is(&ErrBudgetExceeded{}, err)) {
		return err
	}

	if ph == phasePreload {
		// First past was a preload; now do the _real_ pass.
		prog.Budget = budget // reset
		return prog.walkAdv(phaseTraverse, n, s, visitFn)
	}

	return nil
}

// walkAdv is the main recursive walk function, called to iterate through
// recursive nodes (root node, maps, lists and new link root nodes).
func (prog Progress) walkAdv(ph phase, n datamodel.Node, s selector.Selector, visitFn AdvVisitFn) error {
	if err := prog.checkNodeBudget(); err != nil {
		return err
	}

	// If we need to interpret this node in an alternative form, reify and replace.
	if rn, rs, err := prog.reify(n, s); err != nil {
		return err
	} else if rn != nil {
		n = rn
		s = rs
	}

	// Call the visit function if necessary.
	if err := prog.visit(ph, n, s, visitFn); err != nil {
		return err
	}

	// If we're handling scalars (e.g. not maps and lists) we can return now.
	switch n.Kind() {
	case datamodel.Kind_Map, datamodel.Kind_List: // continue
	default:
		return nil
	}

	// For maps and lists: recurse (in one of two ways, depending on if the selector also states specific interests).

	haveStartAtPath := prog.Cfg.StartAtPath.Len() > 0
	var reachedStartAtPath bool
	recurse := func(v datamodel.Node, ps datamodel.PathSegment) error {
		// First, make sure we're past the start path; if one is specified.
		if haveStartAtPath {
			if reachedStartAtPath {
				prog.PastStartAtPath = reachedStartAtPath
			} else if !prog.PastStartAtPath && prog.Path.Len() < prog.Cfg.StartAtPath.Len() {
				if ps.Equals(prog.Cfg.StartAtPath.Segments()[prog.Path.Len()]) {
					reachedStartAtPath = true
				}
				if !reachedStartAtPath {
					return nil
				}
			}
		}

		if err := prog.explore(ph, s, n, visitFn, v, ps); err != nil {
			return err
		}

		return nil
	}

	attn := s.Interests()

	if attn == nil { // no specific interests; recurse on all children.
		for itr := selector.NewSegmentIterator(n); !itr.Done(); {
			ps, v, err := itr.Next()
			if err != nil {
				return err
			}
			if err := recurse(v, ps); err != nil {
				return err
			}
		}

		return nil
	}

	if len(attn) == 0 { // nothing to see here
		return nil
	}

	// specific interests, recurse on those.
	for _, ps := range attn {
		if v, err := n.LookupBySegment(ps); err != nil {
			continue
		} else if err := recurse(v, ps); err != nil {
			return err
		}
	}

	return nil
}

func (prog Progress) checkNodeBudget() error {
	if prog.Budget != nil {
		if prog.Budget.NodeBudget <= 0 {
			return &ErrBudgetExceeded{BudgetKind: "node", Path: prog.Path}
		}
		prog.Budget.NodeBudget--
	}
	return nil
}

func (prog Progress) checkLinkBudget(lnk datamodel.Link) error {
	if prog.Budget != nil {
		if prog.Budget.LinkBudget <= 0 {
			return &ErrBudgetExceeded{BudgetKind: "link", Path: prog.Path, Link: lnk}
		}
		prog.Budget.LinkBudget--
	}
	return nil
}

func (prog Progress) reify(n datamodel.Node, s selector.Selector) (datamodel.Node, selector.Selector, error) {
	// refiy the node if advised.
	if rs, ok := s.(selector.Reifiable); ok {
		adl := rs.NamedReifier()
		if prog.Cfg.LinkSystem.KnownReifiers == nil {
			return nil, nil, fmt.Errorf("adl requested but not supported by link system: %q", adl)
		}

		reifier, ok := prog.Cfg.LinkSystem.KnownReifiers[adl]
		if !ok {
			return nil, nil, fmt.Errorf("unregistered adl requested: %q", adl)
		}

		rn, err := reifier(linking.LinkContext{
			Ctx:      prog.Cfg.Ctx,
			LinkPath: prog.Path,
		}, n, &prog.Cfg.LinkSystem)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to reify node as %q: %w", adl, err)
		}

		// explore into the `InterpretAs` clause to the child selector.
		s, err = s.Explore(n, datamodel.PathSegment{})
		if err != nil {
			return nil, nil, err
		}
		return rn, s, nil
	}

	return nil, nil, nil
}

// visit calls the visitor if required
func (prog Progress) visit(ph phase, n datamodel.Node, s selector.Selector, visitFn AdvVisitFn) error {
	if ph != phaseTraverse {
		return nil
	}

	if !prog.PastStartAtPath && prog.Path.Len() < prog.Cfg.StartAtPath.Len() {
		return nil
	}

	// Decide if this node is matched -- do callbacks as appropriate.
	match, err := s.Match(n)
	if err != nil {
		return err
	}
	if match != nil {
		return visitFn(prog, match, VisitReason_SelectionMatch)
	}
	return visitFn(prog, n, VisitReason_SelectionCandidate)
}

// explore is called to explore a single node, and recurse into it if necessary,
// including loading and recursing into links if the node is a link.
func (prog Progress) explore(
	ph phase,
	s selector.Selector,
	n datamodel.Node,
	visitFn AdvVisitFn,
	v datamodel.Node,
	ps datamodel.PathSegment,
) error {
	sNext, err := s.Explore(n, ps)
	if err != nil {
		return err
	}
	if sNext == nil {
		return nil
	}

	progNext := prog
	progNext.Path = prog.Path.AppendSegment(ps)

	if v.Kind() != datamodel.Kind_Link {
		return progNext.walkAdv(ph, v, sNext, visitFn)
	}

	lnk, _ := v.AsLink()
	if prog.Cfg.LinkVisitOnlyOnce {
		if _, seen := prog.SeenLinks[lnk]; seen {
			return nil
		}
		if ph == phaseTraverse {
			prog.SeenLinks[lnk] = struct{}{}
		}
	}

	if ph == phasePreload {
		if err := prog.checkLinkBudget(lnk); err != nil {
			return err
		}
		pctx := preload.PreloadContext{
			Ctx:        prog.Cfg.Ctx,
			BasePath:   prog.Path,
			ParentNode: n,
		}
		pl := preload.Link{
			Segment:  ps,
			LinkNode: v,
			Link:     lnk,
		}
		prog.Cfg.Preloader(pctx, pl)
		return nil
	}

	progNext.LastBlock.Path = progNext.Path
	progNext.LastBlock.Link = lnk

	v, err = progNext.loadLink(lnk, v, n)
	if err != nil {
		if _, ok := err.(SkipMe); ok {
			return nil
		}
		return err
	}

	return progNext.walkBlock(v, sNext, visitFn)
}

// loadLink is called to load a link from the configured LinkSystem with the
// appropriate prototype.
func (prog Progress) loadLink(lnk datamodel.Link, v datamodel.Node, parent datamodel.Node) (datamodel.Node, error) {
	if err := prog.checkLinkBudget(lnk); err != nil {
		return nil, err
	}
	// Put together the context info we'll offer to the loader and prototypeChooser.
	lnkCtx := linking.LinkContext{
		Ctx:        prog.Cfg.Ctx,
		LinkPath:   prog.Path,
		LinkNode:   v,
		ParentNode: parent,
	}
	// Pick what in-memory format we will build.
	np, err := prog.Cfg.LinkTargetNodePrototypeChooser(lnk, lnkCtx)
	if err != nil {
		return nil, fmt.Errorf("error traversing node at %q: could not load link %q: %w", prog.Path, lnk, err)
	}
	// Load link!
	n, err := prog.Cfg.LinkSystem.Load(lnkCtx, lnk, np)
	if err != nil {
		if _, ok := err.(SkipMe); ok {
			return nil, err
		}
		return nil, fmt.Errorf("error traversing node at %q: could not load link %q: %w", prog.Path, lnk, err)
	}
	return n, nil
}

// WalkTransforming walks a graph of Nodes, deciding which to alter by applying a Selector,
// and calls the given TransformFn to decide what new node to replace the visited node with.
// A new Node tree will be returned (the original is unchanged).
//
// If the TransformFn returns the same Node which it was called with,
// then the transform is a no-op; if every visited node is a no-op,
// then the root node returned from the walk as a whole will also be
// the same as its starting Node (no new memory will be used).
//
// When a Node is replaced, no further recursion of this walk will occur on its contents.
// (You can certainly do a additional traversals, including transforms,
// from inside the TransformFn while building the replacement node.)
//
// The prototype (that is, implementation) of Node returned will be the same as the
// prototype of the Nodes at the same positions in the existing tree
// (literally, builders used to construct any new needed intermediate nodes
// are chosen by asking the existing nodes about their prototype).
func (prog Progress) WalkTransforming(n datamodel.Node, s selector.Selector, fn TransformFn) (datamodel.Node, error) {
	prog.init()
	return prog.walkTransforming(n, s, fn)
}

func (prog Progress) walkTransforming(n datamodel.Node, s selector.Selector, fn TransformFn) (datamodel.Node, error) {
	if err := prog.checkNodeBudget(); err != nil {
		return nil, err
	}

	if rn, rs, err := prog.reify(n, s); err != nil {
		return nil, err
	} else if rn != nil {
		n = rn
		s = rs
	}

	// Decide if this node is matched -- do callbacks as appropriate.
	if s.Decide(n) {
		new_n, err := fn(prog, n)
		if err != nil {
			return nil, err
		}
		if new_n != n {
			// don't continue on transformed subtrees
			return new_n, nil
		}
	}

	// If we're handling scalars (e.g. not maps and lists) we can return now.
	nk := n.Kind()
	switch nk {
	case datamodel.Kind_List:
		return prog.walk_transform_iterateList(n, s, fn, s.Interests())
	case datamodel.Kind_Map:
		return prog.walk_transform_iterateMap(n, s, fn, s.Interests())
	default:
		return n, nil
	}
}

func contains(interest []datamodel.PathSegment, candidate datamodel.PathSegment) bool {
	for _, i := range interest {
		if i == candidate {
			return true
		}
	}
	return false
}

func (prog Progress) walk_transform_iterateList(n datamodel.Node, s selector.Selector, fn TransformFn, attn []datamodel.PathSegment) (datamodel.Node, error) {
	bldr := n.Prototype().NewBuilder()
	lstBldr, err := bldr.BeginList(n.Length())
	if err != nil {
		return nil, err
	}
	for itr := selector.NewSegmentIterator(n); !itr.Done(); {
		ps, v, err := itr.Next()
		if err != nil {
			return nil, err
		}
		if attn == nil || contains(attn, ps) {
			sNext, err := s.Explore(n, ps)
			if err != nil {
				return nil, err
			}
			if sNext != nil {
				progNext := prog
				progNext.Path = prog.Path.AppendSegment(ps)
				if v.Kind() == datamodel.Kind_Link {
					lnk, _ := v.AsLink()
					if prog.Cfg.LinkVisitOnlyOnce {
						if _, seen := prog.SeenLinks[lnk]; seen {
							continue
						}
						prog.SeenLinks[lnk] = struct{}{}
					}
					progNext.LastBlock.Path = progNext.Path
					progNext.LastBlock.Link = lnk
					v, err = progNext.loadLink(lnk, v, n)
					if err != nil {
						if _, ok := err.(SkipMe); ok {
							continue
						}
						return nil, err
					}
				}

				next, err := progNext.WalkTransforming(v, sNext, fn)
				if err != nil {
					return nil, err
				}
				if err := lstBldr.AssembleValue().AssignNode(next); err != nil {
					return nil, err
				}
			} else {
				if err := lstBldr.AssembleValue().AssignNode(v); err != nil {
					return nil, err
				}
			}
		} else {
			if err := lstBldr.AssembleValue().AssignNode(v); err != nil {
				return nil, err
			}
		}
	}
	if err := lstBldr.Finish(); err != nil {
		return nil, err
	}
	return bldr.Build(), nil
}

func (prog Progress) walk_transform_iterateMap(n datamodel.Node, s selector.Selector, fn TransformFn, attn []datamodel.PathSegment) (datamodel.Node, error) {
	bldr := n.Prototype().NewBuilder()
	mapBldr, err := bldr.BeginMap(n.Length())
	if err != nil {
		return nil, err
	}

	for itr := selector.NewSegmentIterator(n); !itr.Done(); {
		ps, v, err := itr.Next()
		if err != nil {
			return nil, err
		}
		if err := mapBldr.AssembleKey().AssignString(ps.String()); err != nil {
			return nil, err
		}

		if attn == nil || contains(attn, ps) {
			sNext, err := s.Explore(n, ps)
			if err != nil {
				return nil, err
			}
			if sNext != nil {
				progNext := prog
				progNext.Path = prog.Path.AppendSegment(ps)
				if v.Kind() == datamodel.Kind_Link {
					lnk, _ := v.AsLink()
					if prog.Cfg.LinkVisitOnlyOnce {
						if _, seen := prog.SeenLinks[lnk]; seen {
							continue
						}
						prog.SeenLinks[lnk] = struct{}{}
					}
					progNext.LastBlock.Path = progNext.Path
					progNext.LastBlock.Link = lnk
					v, err = progNext.loadLink(lnk, v, n)
					if err != nil {
						if _, ok := err.(SkipMe); ok {
							continue
						}
						return nil, err
					}
				}

				next, err := progNext.WalkTransforming(v, sNext, fn)
				if err != nil {
					return nil, err
				}
				if err := mapBldr.AssembleValue().AssignNode(next); err != nil {
					return nil, err
				}
			} else {
				if err := mapBldr.AssembleValue().AssignNode(v); err != nil {
					return nil, err
				}
			}
		} else {
			if err := mapBldr.AssembleValue().AssignNode(v); err != nil {
				return nil, err
			}
		}
	}
	if err := mapBldr.Finish(); err != nil {
		return nil, err
	}
	return bldr.Build(), nil
}
