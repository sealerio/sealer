package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &structReprStringjoinGenerator{}

func NewStructReprStringjoinGenerator(pkgName string, typ *schema.TypeStruct, adjCfg *AdjunctCfg) TypeGenerator {
	return structReprStringjoinGenerator{
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

type structReprStringjoinGenerator struct {
	structGenerator
}

func (g structReprStringjoinGenerator) GetRepresentationNodeGen() NodeGenerator {
	return structReprStringjoinReprGenerator{
		g.AdjCfg,
		mixins.StringTraits{
			PkgName:    g.PkgName,
			TypeName:   string(g.Type.Name()) + ".Repr",
			TypeSymbol: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type structReprStringjoinReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.StringTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprStringjoinReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprStringjoinReprGenerator) EmitNodeType(w io.Writer) {
	// The type is structurally the same, but will have a different set of methods.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}

func (g structReprStringjoinReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g structReprStringjoinReprGenerator) EmitNodeMethodAsString(w io.Writer) {
	// Prerequisites:
	//  - every field must be a string, or have string representation.
	//    - this should've been checked when compiling the type system info.
	//    - we're willing to imply a base-10 atoi/itoa for ints (but it's not currently supported).
	//  - there are NO sanity checks that your value doesn't contain the delimiter
	//    - you need to do this in validation hooks or some other way
	//  - optional or nullable fields are not supported with this representation strategy.
	//    - this should've been checked when compiling the type system info.
	//    - if support for this is added in the future, you can bet all optionals
	//      will be required to be *either* in a row at the start, or in a row at the end.
	//      (a 'direction' property might also be needed, so behavior is defined if every field is optional.)
	//
	// A speciated String method is also generated here.
	//  (Organization questionable: if this was at type level, it'd be in the 'EmitNativeAccessors' block,
	//   but we don't have that in the NodeGenerator interface so we don't have it here.  Maybe that's a mistake.)
	//
	// A String method is *also* generated on the type-level node.
	//  This might be worth consistency review...
	//  It's a practical necessity in areas like stringifying for key error messages if used in map keys, for example.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) AsString() (string, error) {
			return n.String(), nil
		}
		func (n *_{{ .Type | TypeSymbol }}__Repr) String() string {
			return {{ "" }}
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			{{- if $i }} + "{{ $type.RepresentationStrategy.GetDelim }}" + {{end -}}
			(*_{{ $field.Type | TypeSymbol }}__Repr)(&n.{{ $field | FieldSymbolLower }}).String()
			{{- end}}
		}
		func (n {{ .Type | TypeSymbol }}) String() string {
			return (*_{{ .Type | TypeSymbol }}__Repr)(n).String()
		}
	`, w, g.AdjCfg, g)
}

func (g structReprStringjoinReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g structReprStringjoinReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g structReprStringjoinReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return structReprStringjoinReprBuilderGenerator{
		g.AdjCfg,
		mixins.StringAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		g.PkgName,
		g.Type,
	}
}

type structReprStringjoinReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.StringAssemblerTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprStringjoinReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprStringjoinReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g structReprStringjoinReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)

	// Generate a single-step construction function -- this is easy to do for a scalar,
	//  and all representations of scalar kind can be expected to have a method like this.
	// The function is attached to the NodePrototype for convenient namespacing;
	//  it needs no new memory, so it would be inappropriate to attach to the builder or assembler.
	// The function is directly used internally by anything else that might involve recursive destructuring on the same scalar kind
	//  (for example, structs using stringjoin strategies that have one of this type as a field, etc).
	// Since we're a representation of scalar kind, and can recurse,
	//  we ourselves presume this plain construction method must also exist for all our members.
	// REVIEW: We could make an immut-safe verion of this and export it on the NodePrototype too, as `FromString(string)`.
	// FUTURE: should engage validation flow.
	doTemplate(`
		func (_{{ .Type | TypeSymbol }}__ReprPrototype) fromString(w *_{{ .Type | TypeSymbol }}, v string) error {
			ss, err := mixins.SplitExact(v, "{{ .Type.RepresentationStrategy.GetDelim }}", {{ len .Type.Fields }})
			if err != nil {
				return schema.ErrUnmatchable{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Reason: err}
			}
			{{- $dot := . -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			if err := (_{{ $field.Type | TypeSymbol }}__ReprPrototype{}).fromString(&w.{{ $field | FieldSymbolLower }}, ss[{{ $i }}]); err != nil {
				return schema.ErrUnmatchable{TypeName:"{{ $dot.PkgName }}.{{ $dot.Type.Name }}.Repr", Reason: err}
			}
			{{- end}}
			return nil
		}
	`, w, g.AdjCfg, g)
}
func (g structReprStringjoinReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {}
	`, w, g.AdjCfg, g)
}
func (g structReprStringjoinReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g structReprStringjoinReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(w io.Writer) {
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) AssignString(v string) error {
			switch *na.m {
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			}
			{{- if .Type | MaybeUsesPtr }}
			if na.w == nil {
				na.w = &_{{ .Type | TypeSymbol }}{}
			}
			{{- end}}
			if err := (_{{ .Type | TypeSymbol }}__ReprPrototype{}).fromString(na.w, v); err != nil {
				return err
			}
			*na.m = schema.Maybe_Value
			return nil
		}
	`, w, g.AdjCfg, g)
}

func (g structReprStringjoinReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) AssignNode(v datamodel.Node) error {
			if v.IsNull() {
				return na.AssignNull()
			}
			if v2, ok := v.(*_{{ .Type | TypeSymbol }}); ok {
				switch *na.m {
				case schema.Maybe_Value, schema.Maybe_Null:
					panic("invalid state: cannot assign into assembler that's already finished")
				}
				{{- if .Type | MaybeUsesPtr }}
				if na.w == nil {
					na.w = v2
					*na.m = schema.Maybe_Value
					return nil
				}
				{{- end}}
				*na.w = *v2
				*na.m = schema.Maybe_Value
				return nil
			}
			if v2, err := v.AsString(); err != nil {
				return err
			} else {
				return na.AssignString(v2)
			}
		}
	`, w, g.AdjCfg, g)
}
func (g structReprStringjoinReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// None for this.
}
