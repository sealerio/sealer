package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type mapGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeMap
}

func (mapGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g mapGenerator) EmitNativeType(w io.Writer) {
	// Maps do double bookkeeping.
	// - 'm' is used for quick lookup.
	// - 't' is used for both for order maintainence, and for allocation amortization for both keys and values.
	// Note that the key in 'm' is *not* a pointer.
	// The value in 'm' is a pointer into 't' (except when it's a maybe; maybes are already pointers).
	doTemplate(`
		{{- if Comments -}}
		// {{ .Type | TypeSymbol }} matches the IPLD Schema type "{{ .Type.Name }}".  It has {{ .Kind }} kind.
		{{- end}}
		type {{ .Type | TypeSymbol }} = *_{{ .Type | TypeSymbol }}
		type _{{ .Type | TypeSymbol }} struct {
			m map[_{{ .Type.KeyType | TypeSymbol }}]{{if .Type.ValueIsNullable }}Maybe{{else}}*_{{end}}{{ .Type.ValueType | TypeSymbol }}
			t []_{{ .Type | TypeSymbol }}__entry
		}
	`, w, g.AdjCfg, g)
	// - address of 'k' is used when we return keys as nodes, such as in iterators.
	//    Having these in the 't' slice above amortizes moving all of them to heap at once,
	//     which makes iterators that have to return them as an interface much (much) lower cost -- no 'runtime.conv*' pain.
	// - address of 'v' is used in map values, to return, and of course also in iterators.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__entry struct {
			k _{{ .Type.KeyType | TypeSymbol }}
			v _{{ .Type.ValueType | TypeSymbol }}{{if .Type.ValueIsNullable }}__Maybe{{end}}
		}
	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNativeAccessors(w io.Writer) {
	// Generate a speciated Lookup as well as LookupMaybe method.
	// The Lookup method returns nil in case of *either* an absent value or a null value,
	//  and so should only be used if the map type doesn't allow nullable keys or if the caller doesn't care about the difference.
	// The LookupMaybe method returns a MaybeT type for the map value,
	//  and is needed if the map allows nullable values and the caller wishes to distinguish between null and absent.
	// (The Lookup method should be preferred for maps that have non-nullable keys, because LookupMaybe may incur additional costs;
	//   boxing something into a maybe when it wasn't already stored that way costs an alloc(!),
	//    and may additionally incur a memcpy if the maybe for the value type doesn't use pointers internally).
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}) Lookup(k {{ .Type.KeyType | TypeSymbol }}) {{ .Type.ValueType | TypeSymbol }} {
			v, exists := n.m[*k]
			if !exists {
				return nil
			}
			{{- if .Type.ValueIsNullable }}
			if v.m == schema.Maybe_Null {
				return nil
			}
			return {{ if not (MaybeUsesPtr .Type.ValueType) }}&{{end}}v.v
			{{- else}}
			return v
			{{- end}}
		}
		func (n *_{{ .Type | TypeSymbol }}) LookupMaybe(k {{ .Type.KeyType | TypeSymbol }}) Maybe{{ .Type.ValueType | TypeSymbol }} {
			v, exists := n.m[*k]
			if !exists {
				return &_{{ .Type | TypeSymbol }}__valueAbsent
			}
			{{- if .Type.ValueIsNullable }}
			return v
			{{- else}}
			return &_{{ .Type.ValueType | TypeSymbol }}__Maybe{
				m: schema.Maybe_Value,
				v: {{ if not (MaybeUsesPtr .Type.ValueType) }}*{{end}}v,
			}
			{{- end}}
		}

		var _{{ .Type | TypeSymbol }}__valueAbsent = _{{ .Type.ValueType | TypeSymbol }}__Maybe{m:schema.Maybe_Absent}
	`, w, g.AdjCfg, g)

	// Generate a speciated iterator.
	//  The main advantage of this over the general datamodel.MapIterator is of course keeping types visible (and concrete, to the compiler's eyes in optimizations, too).
	//  It also elides the error return from the iterator's Next method.  (Overreads will result in nil keys; this is both easily avoidable, and unambiguous if you do goof and hit it.)
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) Iterator() *{{ .Type | TypeSymbol }}__Itr {
			return &{{ .Type | TypeSymbol }}__Itr{n, 0}
		}

		type {{ .Type | TypeSymbol }}__Itr struct {
			n {{ .Type | TypeSymbol }}
			idx  int
		}

		func (itr *{{ .Type | TypeSymbol }}__Itr) Next() (k {{ .Type.KeyType | TypeSymbol }}, v {{if .Type.ValueIsNullable }}Maybe{{end}}{{ .Type.ValueType | TypeSymbol }}) {
			if itr.idx >= len(itr.n.t) {
				return nil, nil
			}
			x := &itr.n.t[itr.idx]
			k = &x.k
			v = &x.v
			itr.idx++
			return
		}
		func (itr *{{ .Type | TypeSymbol }}__Itr) Done() bool {
			return itr.idx >= len(itr.n.t)
		}

	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNativeBuilder(w io.Writer) {
	// Not yet clear what exactly might be most worth emitting here.
}

func (g mapGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g mapGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g mapGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g mapGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
}

func (g mapGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	// What should be coercible in which directions (and how surprising that is) is an interesting question.
	//  Most of the answer comes from considering what needs to be possible when working with PathSegment:
	//   we *must* be able to accept a string in a PathSegment and be able to use it to navigate a map -- even if the map has complex keys.
	//   For that to work out, it means if the key type doesn't have a string type kind, we must be willing to reach into its representation and use the fromString there.
	//  If the key type *does* have a string kind at the type level, we'll use that; no need to consider going through the representation.
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByString(k string) (datamodel.Node, error) {
			var k2 _{{ .Type.KeyType | TypeSymbol }}
			{{- if eq .Type.KeyType.TypeKind.String "String" }}
			if err := (_{{ .Type.KeyType | TypeSymbol }}__Prototype{}).fromString(&k2, k); err != nil {
				return nil, err // TODO wrap in some kind of ErrInvalidKey
			}
			{{- else}}
			if err := (_{{ .Type.KeyType | TypeSymbol }}__ReprPrototype{}).fromString(&k2, k); err != nil {
				return nil, err // TODO wrap in some kind of ErrInvalidKey
			}
			{{- end}}
			v, exists := n.m[k2]
			if !exists {
				return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(k)}
			}
			{{- if .Type.ValueIsNullable }}
			if v.m == schema.Maybe_Null {
				return datamodel.Null, nil
			}
			return {{ if not (MaybeUsesPtr .Type.ValueType) }}&{{end}}v.v, nil
			{{- else}}
			return v, nil
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	// LookupByNode will procede by cast if it can; or simply error if that doesn't work.
	//  There's no attempt to turn the node (or its repr) into a string and then reify that into a key;
	//   if you used a Node here, you should've meant it.
	// REVIEW: by comparison structs will coerce anything stringish silently...!  so we should figure out if that inconsistency is acceptable, and at least document it if so.
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByNode(k datamodel.Node) (datamodel.Node, error) {
			k2, ok := k.({{ .Type.KeyType | TypeSymbol }})
			if !ok {
				panic("todo invalid key type error")
				// 'schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Key:&_String{k}}' doesn't quite cut it: need room to explain the type, and it's not guaranteed k can be turned into a string at all
			}
			v, exists := n.m[*k2]
			if !exists {
				return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(k2.String())}
			}
			{{- if .Type.ValueIsNullable }}
			if v.m == schema.Maybe_Null {
				return datamodel.Null, nil
			}
			return {{ if not (MaybeUsesPtr .Type.ValueType) }}&{{end}}v.v, nil
			{{- else}}
			return v, nil
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) MapIterator() datamodel.MapIterator {
			return &_{{ .Type | TypeSymbol }}__MapItr{n, 0}
		}

		type _{{ .Type | TypeSymbol }}__MapItr struct {
			n {{ .Type | TypeSymbol }}
			idx  int
		}

		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Next() (k datamodel.Node, v datamodel.Node, _ error) {
			if itr.idx >= len(itr.n.t) {
				return nil, nil, datamodel.ErrIteratorOverread{}
			}
			x := &itr.n.t[itr.idx]
			k = &x.k
			{{- if .Type.ValueIsNullable }}
			switch x.v.m {
			case schema.Maybe_Null:
				v = datamodel.Null
			case schema.Maybe_Value:
				v = {{ if not (MaybeUsesPtr .Type.ValueType) }}&{{end}}x.v.v
			}
			{{- else}}
			v = &x.v
			{{- end}}
			itr.idx++
			return
		}
		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Done() bool {
			return itr.idx >= len(itr.n.t)
		}

	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) Length() int64 {
			return int64(len(n.t))
		}
	`, w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g mapGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g mapGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return mapBuilderGenerator{
		g.AdjCfg,
		mixins.MapAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type mapBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeMap
}

func (mapBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g mapBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
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
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Assembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState

			cm schema.Maybe
			ka _{{ .Type.KeyType | TypeSymbol }}__Assembler
			va _{{ .Type.ValueType | TypeSymbol }}__Assembler
		}

		func (na *_{{ .Type | TypeSymbol }}__Assembler) reset() {
			na.state = maState_initial
			na.ka.reset()
			na.va.reset()
		}
	`, w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_mapoid(w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_mapoid(w, g.AdjCfg, g)
}
func (g mapBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	emitNodeAssemblerHelper_mapoid_keyTidyHelper(w, g.AdjCfg, g)
	emitNodeAssemblerHelper_mapoid_valueTidyHelper(w, g.AdjCfg, g)
	emitNodeAssemblerHelper_mapoid_mapAssemblerMethods(w, g.AdjCfg, g)
}
