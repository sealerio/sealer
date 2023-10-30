package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &listReprListGenerator{}

func NewListReprListGenerator(pkgName string, typ *schema.TypeList, adjCfg *AdjunctCfg) TypeGenerator {
	return listReprListGenerator{
		listGenerator{
			adjCfg,
			mixins.ListTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type listReprListGenerator struct {
	listGenerator
}

func (g listReprListGenerator) GetRepresentationNodeGen() NodeGenerator {
	return listReprListReprGenerator{
		g.AdjCfg,
		mixins.ListTraits{
			PkgName:    g.PkgName,
			TypeName:   string(g.Type.Name()) + ".Repr",
			TypeSymbol: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type listReprListReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.ListTraits
	PkgName string
	Type    *schema.TypeList
}

func (listReprListReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g listReprListReprGenerator) EmitNodeType(w io.Writer) {
	// Even though this is a "natural" representation... we need a new type here,
	//  because lists are recursive, and so all our functions that access
	//   children need to remember to return the representation node of those child values.
	// It's still structurally the same, though (and we'll be able to cast in the methodset pattern).
	// Error-thunking methods also have a different string in their error, so those are unique even if they don't seem particularly interesting.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	// Null is also already a branch in the method we're calling; hopefully the compiler inlines and sees this and DTRT.
	// REVIEW: these unchecked casts are definitely safe at compile time, but I'm not sure if the compiler considers that provable,
	//  so we should investigate if there's any runtime checks injected here that waste time.  If so: write this with more gsloc to avoid :(
	doTemplate(`
		func (nr *_{{ .Type | TypeSymbol }}__Repr) LookupByNode(k datamodel.Node) (datamodel.Node, error) {
			v, err := ({{ .Type | TypeSymbol }})(nr).LookupByNode(k)
			if err != nil || v == datamodel.Null {
				return v, err
			}
			return v.({{ .Type.ValueType | TypeSymbol}}).Representation(), nil
		}
	`, w, g.AdjCfg, g)

}

func (g listReprListReprGenerator) EmitNodeMethodLookupByIndex(w io.Writer) {
	doTemplate(`
		func (nr *_{{ .Type | TypeSymbol }}__Repr) LookupByIndex(idx int64) (datamodel.Node, error) {
			v, err := ({{ .Type | TypeSymbol }})(nr).LookupByIndex(idx)
			if err != nil || v == datamodel.Null {
				return v, err
			}
			return v.({{ .Type.ValueType | TypeSymbol}}).Representation(), nil
		}
	`, w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodeMethodListIterator(w io.Writer) {
	// FUTURE: trying to get this to share the preallocated memory if we get iterators wedged into their node slab will be ... fun.
	doTemplate(`
		func (nr *_{{ .Type | TypeSymbol }}__Repr) ListIterator() datamodel.ListIterator {
			return &_{{ .Type | TypeSymbol }}__ReprListItr{({{ .Type | TypeSymbol }})(nr), 0}
		}

		type _{{ .Type | TypeSymbol }}__ReprListItr _{{ .Type | TypeSymbol }}__ListItr

		func (itr *_{{ .Type | TypeSymbol }}__ReprListItr) Next() (idx int64, v datamodel.Node, err error) {
			idx, v, err = (*_{{ .Type | TypeSymbol }}__ListItr)(itr).Next()
			if err != nil || v == datamodel.Null {
				return
			}
			return idx, v.({{ .Type.ValueType | TypeSymbol}}).Representation(), nil
		}
		func (itr *_{{ .Type | TypeSymbol }}__ReprListItr) Done() bool {
			return (*_{{ .Type | TypeSymbol }}__ListItr)(itr).Done()
		}

	`, w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func (rn *_{{ .Type | TypeSymbol }}__Repr) Length() int64 {
			return int64(len(rn.x))
		}
	`, w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g listReprListReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g listReprListReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return listReprListReprBuilderGenerator{
		g.AdjCfg,
		mixins.ListAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type listReprListReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.ListAssemblerTraits
	PkgName string
	Type    *schema.TypeList
}

func (listReprListReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g listReprListReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the **m**aybe which communicates our completeness to the parent if we're a child assembler.
	// - 'state' is what it says on the tin.  this is used for the list state (the broad transitions between null, start-list, and finish are handled by 'm' for consistency with other types).
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children.
	//    It's only present if list values *aren't* allowed to be nullable, since otherwise they have their own per-value maybe slot we can use.
	// - 'va' is the embedded child value assembler.
	//
	// Note that this textually similar to the type-level assembler, but because it embeds the repr assembler for the child types,
	//  it might be *significantly* different in size and memory layout in that trailing part of the struct.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state laState

			{{ if not .Type.ValueIsNullable }}cm schema.Maybe{{end}}
			va _{{ .Type.ValueType | TypeSymbol }}__ReprAssembler
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			na.state = laState_initial
			na.va.reset()
		}
	`, w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	emitNodeAssemblerMethodBeginList_listoid(w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_listoid(w, g.AdjCfg, g)
}
func (g listReprListReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	emitNodeAssemblerHelper_listoid_tidyHelper(w, g.AdjCfg, g)
	emitNodeAssemblerHelper_listoid_listAssemblerMethods(w, g.AdjCfg, g)
}
