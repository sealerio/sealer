package selectorvalidator

import (
	"errors"

	ipld "github.com/ipld/go-ipld-prime"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

var (
	// ErrInvalidLimit means this type of recursive selector limit is not supported by default
	// -- to prevent DDOS attacks
	ErrInvalidLimit = errors.New("unsupported recursive selector limit")
)

var maxDepthSelector selector.Selector

func init() {
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Map)

	// this selector is a selector for traversing selectors...
	// it traverses the various selector types looking for recursion limit fields
	// and matches them
	maxDepthSelector, _ = ssb.ExploreRecursive(selector.RecursionLimitNone(), ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
		efsb.Insert(selector.SelectorKey_ExploreRecursive, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Limit, ssb.Matcher())
			efsb.Insert(selector.SelectorKey_Sequence, ssb.ExploreRecursiveEdge())
		}))
		efsb.Insert(selector.SelectorKey_ExploreFields, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Fields, ssb.ExploreAll(ssb.ExploreRecursiveEdge()))
		}))
		efsb.Insert(selector.SelectorKey_ExploreUnion, ssb.ExploreAll(ssb.ExploreRecursiveEdge()))
		efsb.Insert(selector.SelectorKey_ExploreAll, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Next, ssb.ExploreRecursiveEdge())
		}))
		efsb.Insert(selector.SelectorKey_ExploreIndex, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Next, ssb.ExploreRecursiveEdge())
		}))
		efsb.Insert(selector.SelectorKey_ExploreRange, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Next, ssb.ExploreRecursiveEdge())
		}))
		efsb.Insert(selector.SelectorKey_ExploreConditional, ssb.ExploreFields(func(efsb builder.ExploreFieldsSpecBuilder) {
			efsb.Insert(selector.SelectorKey_Next, ssb.ExploreRecursiveEdge())
		}))
	})).Selector()
}

// SelectorValidator returns an OnRequestReceivedHook that only validates
// requests if their selector only has no recursions that are greater than
// maxAcceptedDepth
func SelectorValidator(maxAcceptedDepth int64) graphsync.OnIncomingRequestHook {
	return func(p peer.ID, request graphsync.RequestData, hookActions graphsync.IncomingRequestHookActions) {
		err := ValidateMaxRecursionDepth(request.Selector(), maxAcceptedDepth)
		if err == nil {
			hookActions.ValidateRequest()
		}
	}
}

// ValidateMaxRecursionDepth examines the given selector node and verifies
// recursive selectors are limited to the given fixed depth
func ValidateMaxRecursionDepth(node ipld.Node, maxAcceptedDepth int64) error {

	return traversal.WalkMatching(node, maxDepthSelector, func(progress traversal.Progress, visited ipld.Node) error {
		if visited.Kind() != ipld.Kind_Map || visited.Length() != 1 {
			return ErrInvalidLimit
		}
		kn, v, _ := visited.MapIterator().Next()
		kstr, _ := kn.AsString()
		switch kstr {
		case selector.SelectorKey_LimitDepth:
			maxDepthValue, err := v.AsInt()
			if err != nil {
				return ErrInvalidLimit
			}
			if maxDepthValue > maxAcceptedDepth {
				return ErrInvalidLimit
			}
			return nil
		case selector.SelectorKey_LimitNone:
			return ErrInvalidLimit
		default:
			return ErrInvalidLimit
		}
	})
}
