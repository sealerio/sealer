package builder

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/fluent"
	"github.com/ipld/go-ipld-prime/traversal/selector"
)

// SelectorSpec is a specification for a selector that can build
// a selector datamodel.Node or an actual parsed Selector
type SelectorSpec interface {
	Node() datamodel.Node
	Selector() (selector.Selector, error)
}

// SelectorSpecBuilder is a utility interface to build selector ipld nodes
// quickly.
//
// It serves two purposes:
// 1. Save the user of go-ipld time and mental overhead with an easy
// interface for making selector nodes in much less code without having to remember
// the selector sigils
// 2. Provide a level of protection from selector schema changes, at least in terms
// of naming, if not structure
type SelectorSpecBuilder interface {
	ExploreRecursiveEdge() SelectorSpec
	ExploreRecursive(limit selector.RecursionLimit, sequence SelectorSpec) SelectorSpec
	ExploreUnion(...SelectorSpec) SelectorSpec
	ExploreAll(next SelectorSpec) SelectorSpec
	ExploreIndex(index int64, next SelectorSpec) SelectorSpec
	ExploreRange(start, end int64, next SelectorSpec) SelectorSpec
	ExploreFields(ExploreFieldsSpecBuildingClosure) SelectorSpec
	ExploreInterpretAs(as string, next SelectorSpec) SelectorSpec
	Matcher() SelectorSpec
	MatcherSubset(from, to int64) SelectorSpec
}

// ExploreFieldsSpecBuildingClosure is a function that provided to SelectorSpecBuilder's
// ExploreFields method that assembles the fields map in the selector using
// an ExploreFieldsSpecBuilder
type ExploreFieldsSpecBuildingClosure func(ExploreFieldsSpecBuilder)

// ExploreFieldsSpecBuilder is an interface for assemble the map of fields to
// selectors in ExploreFields
type ExploreFieldsSpecBuilder interface {
	Insert(k string, v SelectorSpec)
}

type selectorSpecBuilder struct {
	np datamodel.NodePrototype
}

type selectorSpec struct {
	n datamodel.Node
}

func (ss selectorSpec) Node() datamodel.Node {
	return ss.n
}

func (ss selectorSpec) Selector() (selector.Selector, error) {
	return selector.ParseSelector(ss.n)
}

// NewSelectorSpecBuilder creates a SelectorSpecBuilder which will store
// data in the format determined by the given datamodel.NodePrototype.
func NewSelectorSpecBuilder(np datamodel.NodePrototype) SelectorSpecBuilder {
	return &selectorSpecBuilder{np}
}

func (ssb *selectorSpecBuilder) ExploreRecursiveEdge() SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreRecursiveEdge).CreateMap(0, func(na fluent.MapAssembler) {})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreRecursive(limit selector.RecursionLimit, sequence SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreRecursive).CreateMap(2, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Limit).CreateMap(1, func(na fluent.MapAssembler) {
					switch limit.Mode() {
					case selector.RecursionLimit_Depth:
						na.AssembleEntry(selector.SelectorKey_LimitDepth).AssignInt(limit.Depth())
					case selector.RecursionLimit_None:
						na.AssembleEntry(selector.SelectorKey_LimitNone).CreateMap(0, func(na fluent.MapAssembler) {})
					default:
						panic("Unsupported recursion limit type")
					}
				})
				na.AssembleEntry(selector.SelectorKey_Sequence).AssignNode(sequence.Node())
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreAll(next SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreAll).CreateMap(1, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Next).AssignNode(next.Node())
			})
		}),
	}
}
func (ssb *selectorSpecBuilder) ExploreIndex(index int64, next SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreIndex).CreateMap(2, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Index).AssignInt(index)
				na.AssembleEntry(selector.SelectorKey_Next).AssignNode(next.Node())
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreRange(start, end int64, next SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreRange).CreateMap(3, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Start).AssignInt(start)
				na.AssembleEntry(selector.SelectorKey_End).AssignInt(end)
				na.AssembleEntry(selector.SelectorKey_Next).AssignNode(next.Node())
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreUnion(members ...SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreUnion).CreateList(int64(len(members)), func(na fluent.ListAssembler) {
				for _, member := range members {
					na.AssembleValue().AssignNode(member.Node())
				}
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreFields(specBuilder ExploreFieldsSpecBuildingClosure) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreFields).CreateMap(1, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Fields).CreateMap(-1, func(na fluent.MapAssembler) {
					specBuilder(exploreFieldsSpecBuilder{na})
				})
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) ExploreInterpretAs(as string, next SelectorSpec) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_ExploreInterpretAs).CreateMap(1, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_As).AssignString(as)
				na.AssembleEntry(selector.SelectorKey_Next).AssignNode(next.Node())
			})
		}),
	}
}

func (ssb *selectorSpecBuilder) Matcher() SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_Matcher).CreateMap(0, func(na fluent.MapAssembler) {})
		}),
	}
}

func (ssb *selectorSpecBuilder) MatcherSubset(from, to int64) SelectorSpec {
	return selectorSpec{
		fluent.MustBuildMap(ssb.np, 1, func(na fluent.MapAssembler) {
			na.AssembleEntry(selector.SelectorKey_Matcher).CreateMap(1, func(na fluent.MapAssembler) {
				na.AssembleEntry(selector.SelectorKey_Subset).CreateMap(2, func(na fluent.MapAssembler) {
					na.AssembleEntry(selector.SelectorKey_From).AssignInt(from)
					na.AssembleEntry(selector.SelectorKey_To).AssignInt(to)
				})
			})
		}),
	}
}

type exploreFieldsSpecBuilder struct {
	na fluent.MapAssembler
}

func (efsb exploreFieldsSpecBuilder) Insert(field string, s SelectorSpec) {
	efsb.na.AssembleEntry(field).AssignNode(s.Node())
}
