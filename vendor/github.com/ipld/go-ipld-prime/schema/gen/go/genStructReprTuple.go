package gengo

import (
	"io"
	"strconv"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &structReprTupleGenerator{}

// Optional fields for tuple representation are only allowed at the end, and contiguously.
// Present fields are matched greedily: if the struct has five fields,
//  and the last two are optional, and there's four values, then they will be mapped onto the first four fields, period.
// In theory, it would be possible to support a variety of fancier modes, configurably;
//  in practice, let's not: the ROI would be atrocious:
//   few people seem to want this;
//   the implementation complexity would rise dramatically;
//   and the next nearest substitutes for such behavior are already available, and cheap (and also sturdier).
// It would make about as much sense to support implicits as it does trailing optionals,
//  which means we probably should consider that someday,
//   but it's not implemented today.

func NewStructReprTupleGenerator(pkgName string, typ *schema.TypeStruct, adjCfg *AdjunctCfg) TypeGenerator {
	return structReprTupleGenerator{
		structGenerator{
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

type structReprTupleGenerator struct {
	structGenerator
}

func (g structReprTupleGenerator) GetRepresentationNodeGen() NodeGenerator {
	return structReprTupleReprGenerator{
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

type structReprTupleReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.ListTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprTupleReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprTupleReprGenerator) EmitNodeType(w io.Writer) {
	// The type is structurally the same, but will have a different set of methods.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodeMethodLookupByIndex(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) LookupByIndex(idx int64) (datamodel.Node, error) {
			switch idx {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsOptional }}
				if n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					return datamodel.Absent, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfInt(idx)}
				}
				{{- end}}
				{{- if $field.IsNullable }}
				if n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Null {
					return datamodel.Null, nil
				}
				{{- end}}
				{{- if $field.IsMaybe }}
				return n.{{ $field | FieldSymbolLower }}.v.Representation(), nil
				{{- else}}
				return n.{{ $field | FieldSymbolLower }}.Representation(), nil
				{{- end}}
			{{- end}}
			default:
				return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfInt(idx)}
			}
		}
	`, w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
			ki, err := key.AsInt()
			if err != nil {
				return nil, err
			}
			return n.LookupByIndex(ki)
		}
	`, w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodeMethodListIterator(w io.Writer) {
	// DRY: much of this precalcuation about doneness is common with the map representation.
	//  (or at least: it is for now: the addition of support for implicits in the map representation may bamboozle that.)
	//  Some of the templating also experiences the `.HaveTrailingOptionals` branching,
	//   but not quite as much as the map representation: since we always know those come at the end
	//    (and in particular, once we hit one absent, we're done!), some simplifications can be made.

	// The 'idx' int is what field we'll yield next.
	// Note that this iterator doesn't mention fields that are absent.
	//  This makes things a bit trickier -- especially the 'Done' predicate,
	//   since it may have to do lookahead if there's any optionals at the end of the structure!

	// Count how many trailing fields are optional.
	//  The 'Done' predicate gets more complex when in the trailing optionals.
	fields := g.Type.Fields()
	fieldCount := len(fields)
	beginTrailingOptionalField := fieldCount
	for i := fieldCount - 1; i >= 0; i-- {
		if !fields[i].IsOptional() {
			break
		}
		beginTrailingOptionalField = i
	}
	haveTrailingOptionals := beginTrailingOptionalField < fieldCount

	// Now: finally we can get on with the actual templating.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) ListIterator() datamodel.ListIterator {
			{{- if .HaveTrailingOptionals }}
			end := {{ len .Type.Fields }}`+
		func() string { // this next part was too silly in templates due to lack of reverse ranging.
			v := "\n"
			for i := fieldCount - 1; i >= beginTrailingOptionalField; i-- {
				v += "\t\t\tif n." + g.AdjCfg.FieldSymbolLower(fields[i]) + ".m == schema.Maybe_Absent {\n"
				v += "\t\t\t\tend = " + strconv.Itoa(i) + "\n"
				v += "\t\t\t} else {\n"
				v += "\t\t\t\tgoto done\n"
				v += "\t\t\t}\n"
			}
			return v
		}()+`done:
			return &_{{ .Type | TypeSymbol }}__ReprListItr{n, 0, end}
			{{- else}}
			return &_{{ .Type | TypeSymbol }}__ReprListItr{n, 0}
			{{- end}}
		}

		type _{{ .Type | TypeSymbol }}__ReprListItr struct {
			n   *_{{ .Type | TypeSymbol }}__Repr
			idx int
			{{if .HaveTrailingOptionals }}end int{{end}}
		}

		func (itr *_{{ .Type | TypeSymbol }}__ReprListItr) Next() (idx int64, v datamodel.Node, err error) {
			if itr.idx >= {{ len .Type.Fields }} {
				return -1, nil, datamodel.ErrIteratorOverread{}
			}
			switch itr.idx {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				idx = int64(itr.idx)
				{{- if $field.IsOptional }}
				if itr.n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					return -1, nil, datamodel.ErrIteratorOverread{}
				}
				{{- end}}
				{{- if $field.IsNullable }}
				if itr.n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Null {
					v = datamodel.Null
					break
				}
				{{- end}}
				{{- if $field.IsMaybe }}
				v = itr.n.{{ $field | FieldSymbolLower}}.v.Representation()
				{{- else}}
				v = itr.n.{{ $field | FieldSymbolLower}}.Representation()
				{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
			itr.idx++
			return
		}
		{{- if .HaveTrailingOptionals }}
		func (itr *_{{ .Type | TypeSymbol }}__ReprListItr) Done() bool {
			return itr.idx >= itr.end
		}
		{{- else}}
		func (itr *_{{ .Type | TypeSymbol }}__ReprListItr) Done() bool {
			return itr.idx >= {{ len .Type.Fields }}
		}
		{{- end}}

	`, w, g.AdjCfg, struct {
		Type                  *schema.TypeStruct
		HaveTrailingOptionals bool
	}{
		g.Type,
		haveTrailingOptionals,
	})
}

func (g structReprTupleReprGenerator) EmitNodeMethodLength(w io.Writer) {
	// This is fun: it has to count down for any unset optional fields.
	doTemplate(`
		func (rn *_{{ .Type | TypeSymbol }}__Repr) Length() int64 {
			l := {{ len .Type.Fields }}
			{{- range $field := .Type.Fields }}
			{{- if $field.IsOptional }}
			if rn.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
				l--
			}
			{{- end}}
			{{- end}}
			return int64(l)
		}
	`, w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g structReprTupleReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g structReprTupleReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return structReprTupleReprBuilderGenerator{
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

type structReprTupleReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.ListAssemblerTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprTupleReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprTupleReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the **m**aybe which communicates our completeness to the parent if we're a child assembler.
	// - 'state' is what it says on the tin.  this is used for the list state (the broad transitions between null, start-list, and finish are handled by 'm' for consistency with other types).
	// - contrasted to the map representation, there's no 's' bitfield for what's been **s**et -- because we know things must procede in order, it would be redundant with 'f'.
	// - 'f' is the **f**ocused field that will be assembled next.
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children that aren't allowed to be nullable (for those that are, their own maybe.m is used).
	// - the 'ca_*' fields embed **c**hild **a**ssemblers -- these are embedded so we can yield pointers to them without causing new allocations.
	//
	// Note that this textually similar to the type-level assembler, but because it embeds the repr assembler for the child types,
	//  it might be *significantly* different in size and memory layout in that trailing part of the struct.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state laState
			f int

			cm schema.Maybe
			{{range $field := .Type.Fields -}}
			ca_{{ $field | FieldSymbolLower }} _{{ $field.Type | TypeSymbol }}__ReprAssembler
			{{end -}}
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			na.state = laState_initial
			na.f = 0
			{{- range $field := .Type.Fields }}
			na.ca_{{ $field | FieldSymbolLower }}.reset()
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	// Future: This could do something strict with the sizehint; it currently ignores it.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) BeginList(int64) (datamodel.ListAssembler, error) {
			switch *na.m {
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			case midvalue:
				panic("invalid state: it makes no sense to 'begin' twice on the same assembler!")
			}
			*na.m = midvalue
			{{- if .Type | MaybeUsesPtr }}
			if na.w == nil {
				na.w = &_{{ .Type | TypeSymbol }}{}
			}
			{{- end}}
			return na, nil
		}
	`, w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_listoid(w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	g.emitListAssemblerChildTidyHelper(w)
	g.emitListAssemblerChildListAssemblerMethods(w)
}
func (g structReprTupleReprBuilderGenerator) emitListAssemblerChildTidyHelper(w io.Writer) {
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__ReprAssembler) valueFinishTidy() bool {
			switch la.f {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsMaybe }}
				switch la.w.{{ $field | FieldSymbolLower }}.m {
				case schema.Maybe_Value:
					{{- if (MaybeUsesPtr $field.Type) }}
					la.w.{{ $field | FieldSymbolLower }}.v = la.ca_{{ $field | FieldSymbolLower }}.w
					{{- end}}
					la.state = laState_initial
					la.f++
					return true
				{{- else}}
				switch la.cm {
				case schema.Maybe_Value:
					la.cm = schema.Maybe_Absent
					la.state = laState_initial
					la.f++
					return true
				{{- end}}
				{{- if $field.IsNullable }}
				case schema.Maybe_Null:
					la.state = laState_initial
					la.f++
					return true
				{{- end}}
				default:
					return false
				}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)
}
func (g structReprTupleReprBuilderGenerator) emitListAssemblerChildListAssemblerMethods(w io.Writer) {
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__ReprAssembler) AssembleValue() datamodel.NodeAssembler {
			switch la.state {
			case laState_initial:
				// carry on
			case laState_midValue:
				if !la.valueFinishTidy() {
					panic("invalid state: AssembleValue cannot be called when still in the middle of assembling the previous value")
				} // if tidy success: carry on
			case laState_finished:
				panic("invalid state: AssembleValue cannot be called on an assembler that's already finished")
			}
			if la.f >= {{ len .Type.Fields }} {
				return _ErrorThunkAssembler{schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfInt({{ len .Type.Fields }})}}
			}
			la.state = laState_midValue
			switch la.f {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsMaybe }}
				la.ca_{{ $field | FieldSymbolLower }}.w = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}la.w.{{ $field | FieldSymbolLower }}.v
				la.ca_{{ $field | FieldSymbolLower }}.m = &la.w.{{ $field | FieldSymbolLower }}.m
				{{- if $field.IsNullable }}
				la.w.{{ $field | FieldSymbolLower }}.m = allowNull
				{{- end}}
				{{- else}}
				la.ca_{{ $field | FieldSymbolLower }}.w = &la.w.{{ $field | FieldSymbolLower }}
				la.ca_{{ $field | FieldSymbolLower }}.m = &la.cm
				{{- end}}
				return &la.ca_{{ $field | FieldSymbolLower }}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)
	// Surprisingly, the Finish method doesn't have anything to do regarding any trailing optionals:
	//  if they weren't assigned yet, their Maybe state is still the zero value: absent.  And that's correct.
	// DRY: okay, this finish component is actually identical, both textually and in terms of linking, to lists.  This we should actually extract.
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__ReprAssembler) Finish() error {
			switch la.state {
			case laState_initial:
				// carry on
			case laState_midValue:
				if !la.valueFinishTidy() {
					panic("invalid state: Finish cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on
			case laState_finished:
				panic("invalid state: Finish cannot be called on an assembler that's already finished")
			}
			la.state = laState_finished
			*la.m = schema.Maybe_Value
			return nil
		}
	`, w, g.AdjCfg, g)
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__ReprAssembler) ValuePrototype(_ int64) datamodel.NodePrototype {
			panic("todo structbuilder tuplerepr valueprototype")
		}
	`, w, g.AdjCfg, g)
}
