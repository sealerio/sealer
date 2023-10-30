package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &unionReprStringprefixGenerator{}

func NewUnionReprStringprefixGenerator(pkgName string, typ *schema.TypeUnion, adjCfg *AdjunctCfg) TypeGenerator {
	return unionReprStringprefixGenerator{
		unionGenerator{
			AdjCfg: adjCfg,
			MapTraits: mixins.MapTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			PkgName: pkgName,
			Type:    typ,
		},
	}
}

type unionReprStringprefixGenerator struct {
	unionGenerator
}

func (g unionReprStringprefixGenerator) GetRepresentationNodeGen() NodeGenerator {
	return unionReprStringprefixReprGenerator{
		AdjCfg: g.AdjCfg,
		StringTraits: mixins.StringTraits{
			PkgName:    g.PkgName,
			TypeName:   string(g.Type.Name()) + ".Repr",
			TypeSymbol: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Repr",
		},
		PkgName: g.PkgName,
		Type:    g.Type,
	}
}

type unionReprStringprefixReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.StringTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionReprStringprefixReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionReprStringprefixReprGenerator) EmitNodeType(w io.Writer) {
	// The type is structurally the same, but will have a different set of methods.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)

	// We do also want some constants for our discriminant values;
	//  they'll make iterators able to work faster.
	doTemplate(`
		var (
			{{- range $member := .Type.Members }}
			memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}_serial = _String{"{{ $member | dot.Type.RepresentationStrategy.GetDiscriminant }}"}
			{{- end }}
		)
	`, w, g.AdjCfg, g)
}

func (g unionReprStringprefixReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g unionReprStringprefixReprGenerator) EmitNodeMethodAsString(w io.Writer) {
	// See comment block in structReprStringjoinReprGenerator.EmitNodeMethodAsString for a lot of philosophizing about this.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) AsString() (string, error) {
			return n.String(), nil
		}
		func (n *_{{ .Type | TypeSymbol }}__Repr) String() string {
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch n.tag {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				return memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}_serial.String() + "{{ dot.Type.RepresentationStrategy.GetDelim }}" + n.x{{ add $i 1 }}.String()
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			switch n2 := n.x.(type) {
			{{- range $member := .Type.Members }}
			case {{ $member | TypeSymbol }}:
				return memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}_serial.String() + "{{ dot.Type.RepresentationStrategy.GetDelim }}" + n2.String()
			{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
		func (n {{ .Type | TypeSymbol }}) String() string {
			return (*_{{ .Type | TypeSymbol }}__Repr)(n).String()
		}
	`, w, g.AdjCfg, g)
}

func (g unionReprStringprefixReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g unionReprStringprefixReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g unionReprStringprefixReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return unionReprStringprefixReprBuilderGenerator{
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

type unionReprStringprefixReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.StringAssemblerTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionReprStringprefixReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionReprStringprefixReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g unionReprStringprefixReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
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
	doTemplate(`
		func (_{{ .Type | TypeSymbol }}__ReprPrototype) fromString(w *_{{ .Type | TypeSymbol }}, v string) error {
			ss := mixins.SplitN(v, "{{ .Type.RepresentationStrategy.GetDelim }}", 2)
			if len(ss) != 2 {
				return schema.ErrUnmatchable{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr"}.Reasonf("expecting a stringprefix union but found no delimiter in the value")
			}
			switch ss[0] {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member | dot.Type.RepresentationStrategy.GetDiscriminant }}":
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				w.tag = {{ add $i 1 }}
				if err := (_{{ $member | TypeSymbol }}__ReprPrototype{}).fromString(&w.x{{ add $i 1 }}, ss[1]); err != nil {
					return schema.ErrUnmatchable{TypeName:"{{ dot.PkgName }}.{{ dot.Type.Name }}.Repr", Reason: err}
				}
				return nil
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				var n2 _{{ $member | TypeSymbol }}
				if err := (_{{ $member | TypeSymbol }}__ReprPrototype{}).fromString(&n2, ss[1]); err != nil {
					return schema.ErrUnmatchable{TypeName:"{{ dot.PkgName }}.{{ dot.Type.Name }}.Repr", Reason: err}
				}
				w.x = &n2
				return nil
				{{- end}}
			{{- end}}
			default:
				return schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfString(ss[0])}
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionReprStringprefixReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {}
	`, w, g.AdjCfg, g)
}
func (g unionReprStringprefixReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g unionReprStringprefixReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(w io.Writer) {
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	// TODO:DRY: this is identical to other string-repr-on-non-string-type.
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

func (g unionReprStringprefixReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	// TODO:DRY: this is identical to other string-repr-on-non-string-type.
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
func (g unionReprStringprefixReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// None for this.
}
