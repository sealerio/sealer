package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &mapReprMapGenerator{}

func NewMapReprMapGenerator(pkgName string, typ *schema.TypeMap, adjCfg *AdjunctCfg) TypeGenerator {
	return mapReprMapGenerator{
		mapGenerator{
			adjCfg,
			mixins.MapTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type mapReprMapGenerator struct {
	mapGenerator
}

func (g mapReprMapGenerator) GetRepresentationNodeGen() NodeGenerator {
	return mapReprMapReprGenerator{
		g.AdjCfg,
		mixins.MapTraits{
			PkgName:    g.PkgName,
			TypeName:   string(g.Type.Name()) + ".Repr",
			TypeSymbol: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type mapReprMapReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeMap
}

func (mapReprMapReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g mapReprMapReprGenerator) EmitNodeType(w io.Writer) {
	// Even though this is a "natural" representation... we need a new type here,
	//  because maps are recursive, and so all our functions that access
	//   children need to remember to return the representation node of those child values.
	// It's still structurally the same, though (and we'll be able to cast in the methodset pattern).
	// Error-thunking methods also have a different string in their error, so those are unique even if they don't seem particularly interesting.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g mapReprMapReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g mapReprMapReprGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	doTemplate(`
		func (nr *_{{ .Type | TypeSymbol }}__Repr) LookupByString(k string) (datamodel.Node, error) {
			v, err := ({{ .Type | TypeSymbol }})(nr).LookupByString(k)
			if err != nil || v == datamodel.Null {
				return v, err
			}
			return v.({{ .Type.ValueType | TypeSymbol}}).Representation(), nil
		}
	`, w, g.AdjCfg, g)
}
func (g mapReprMapReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
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
func (g mapReprMapReprGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	// FUTURE: trying to get this to share the preallocated memory if we get iterators wedged into their node slab will be ... fun.
	doTemplate(`
		func (nr *_{{ .Type | TypeSymbol }}__Repr) MapIterator() datamodel.MapIterator {
			return &_{{ .Type | TypeSymbol }}__ReprMapItr{({{ .Type | TypeSymbol }})(nr), 0}
		}

		type _{{ .Type | TypeSymbol }}__ReprMapItr _{{ .Type | TypeSymbol }}__MapItr

		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Next() (k datamodel.Node, v datamodel.Node, err error) {
			k, v, err = (*_{{ .Type | TypeSymbol }}__MapItr)(itr).Next()
			if err != nil || v == datamodel.Null {
				return
			}
			return k, v.({{ .Type.ValueType | TypeSymbol}}).Representation(), nil
		}
		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Done() bool {
			return (*_{{ .Type | TypeSymbol }}__MapItr)(itr).Done()
		}

	`, w, g.AdjCfg, g)
}
func (g mapReprMapReprGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func (rn *_{{ .Type | TypeSymbol }}__Repr) Length() int64 {
			return int64(len(rn.t))
		}
	`, w, g.AdjCfg, g)
}
func (g mapReprMapReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}
func (g mapReprMapReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g mapReprMapReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return mapReprMapReprBuilderGenerator{
		g.AdjCfg,
		mixins.MapAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type mapReprMapReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeMap
}

func (mapReprMapReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g mapReprMapReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the **m**aybe which communicates our completeness to the parent if we're a child assembler.
	// - 'state' is what it says on the tin.  this is used for the map state (the broad transitions between null, start-map, and finish are handled by 'm' for consistency.)
	// - there's no equivalent of the 'f' (**f**ocused next) field in struct assemblers -- that's implicitly the last row of the 'w.t'.
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children.
	//    It's used for values if values aren't allowed to be nullable and thus don't have their own per-value maybe slot we can use.
	//    It's always used for key assembly, since keys are never allowed to be nullable and thus etc.
	// - 'ka' and 'va' are the key assembler and value assembler respectively.
	//    Perhaps surprisingly, we can get away with using the assemblers for each type just straight up, no wrappers necessary;
	//     All of the required magic is handled through maybe pointers and some tidy methods used during state transitions.
	//
	// Note that this textually similar to the type-level assembler, but because it embeds the repr assembler for the child types,
	//  it might be *significantly* different in size and memory layout in that trailing part of the struct.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState

			cm schema.Maybe
			ka _{{ .Type.KeyType | TypeSymbol }}__ReprAssembler
			va _{{ .Type.ValueType | TypeSymbol }}__ReprAssembler
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			na.state = maState_initial
			na.ka.reset()
			na.va.reset()
		}
	`, w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_mapoid(w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_mapoid(w, g.AdjCfg, g)
}
func (g mapReprMapReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	emitNodeAssemblerHelper_mapoid_keyTidyHelper(w, g.AdjCfg, g)
	emitNodeAssemblerHelper_mapoid_valueTidyHelper(w, g.AdjCfg, g)
	emitNodeAssemblerHelper_mapoid_mapAssemblerMethods(w, g.AdjCfg, g)
}
