package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &unionKindedGenerator{}

// Kinded union representations are quite wild: their behavior varies almost completely per inhabitant,
//  and their implementation is generally delegating directly to something else,
//   rather than having an intermediate node (like most unions do, and like the type-level view of this same value will).
//
// This also means any error values can be a little weird:
//  sometimes they'll have the union's type name, but sometimes they'll have the inhabitant's type name instead;
//  this depends on whether the error is an ErrWrongKind that was found while checking the method for appropriateness on the union's inhabitant
//  versus if the error came from the union inhabitant itself after delegation occured.

func NewUnionReprKindedGenerator(pkgName string, typ *schema.TypeUnion, adjCfg *AdjunctCfg) TypeGenerator {
	return unionKindedGenerator{
		unionGenerator{
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

type unionKindedGenerator struct {
	unionGenerator
}

func (g unionKindedGenerator) GetRepresentationNodeGen() NodeGenerator {
	return unionKindedReprGenerator{
		g.AdjCfg,
		g.PkgName,
		g.Type,
	}
}

type unionKindedReprGenerator struct {
	// Note that there's no MapTraits (or any other FooTraits) mixin in this one!
	//  This is no accident: *None* of them apply!

	AdjCfg  *AdjunctCfg
	PkgName string
	Type    *schema.TypeUnion
}

func (unionKindedReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionKindedReprGenerator) EmitNodeType(w io.Writer) {
	// The type is structurally the same, but will have a different set of methods.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodKind(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) Kind() datamodel.Kind {
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch n.tag {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				return {{ $member.RepresentationBehavior | KindSymbol }}
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			switch n.x.(type) {
			{{- range $i, $member := .Type.Members }}
			case {{ $member | TypeSymbol }}:
				return {{ $member.RepresentationBehavior | KindSymbol }}
			{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)
}

func kindedUnionNodeMethodTemplateMunge(
	methodName string, // for error messages
	methodSig string, // output literally
	someSwitchClause string, // template condition for if *any* switch clause should be present
	condClause string, // template condition for the member this should match on when in the range
	retClause string, // clause returning the thing (how to delegate methodsig, generally)
	appropriateKind string, // for error messages
	nopeSentinel string, // for error return paths; generally the zero value for the first return type.
	nopeSentinelOnly bool, // true if this method has no error return, just the sentinel.
) string {
	// We really could just... call the methods directly (and elide the switch entirely all the time), in the case of the "interface" implementation strategy.
	//  We don't, though, because that would deprive us of getting the union type's name in the wrong-kind errors...
	//   and in addition to that being sadface in general, it would be downright unacceptable if that behavior varied based on implementation strategy.
	//
	// This error text doesn't tell us what the member kind is.  This might read weirdly.
	//  It's possible we could try to cram a description of the inhabitant into the "TypeName" since it's stringy; but unclear if that's a good idea either.

	// These template concatenations have evolved into a mess very quickly.  This entire thing should be replaced.
	//  String concatenations of template clauses is an atrociously unhygenic way to compose things;
	//   it looked like we could limp by with it for a while, but it's gotten messier faster than expected.

	errorClause := `return ` + nopeSentinel
	if !nopeSentinelOnly {
		errorClause += `, datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}.Repr", MethodName: "` + methodName + `", AppropriateKind: ` + appropriateKind + `, ActualKind: n.Kind()}`
	}
	return `
		func (n *_{{ .Type | TypeSymbol }}__Repr) ` + methodSig + ` {
			` + someSwitchClause + `
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch n.tag {
			{{- range $i, $member := .Type.Members }}
			` + condClause + `
			case {{ add $i 1 }}:
				return n.x{{ add $i 1 }}.Representation()` + retClause + `
			{{- end}}
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			switch n2 := n.x.(type) {
			{{- range $i, $member := .Type.Members }}
			` + condClause + `
			case {{ $member | TypeSymbol }}:
				return n2.Representation()` + retClause + `
			{{- end}}
			{{- end}}
			{{- end}}
			default:
			{{- end}}
				` + errorClause + `
			` + someSwitchClause + `
			}
			{{- end}}
		}
	`
}

func (g unionKindedReprGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`LookupByString`,
		`LookupByString(key string) (datamodel.Node, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "map") }}`,
		`{{- if eq $member.RepresentationBehavior.String "map" }}`,
		`.LookupByString(key)`,
		`datamodel.KindSet_JustMap`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodLookupByIndex(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`LookupByIndex`,
		`LookupByIndex(idx int64) (datamodel.Node, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "list") }}`,
		`{{- if eq $member.RepresentationBehavior.String "list" }}`,
		`.LookupByIndex(idx)`,
		`datamodel.KindSet_JustList`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`LookupByNode`,
		`LookupByNode(key datamodel.Node) (datamodel.Node, error)`,
		`{{- if or (.Type.RepresentationStrategy.GetMember (Kind "map")) (.Type.RepresentationStrategy.GetMember (Kind "list")) }}`,
		`{{- if or (eq $member.RepresentationBehavior.String "map") (eq $member.RepresentationBehavior.String "list") }}`,
		`.LookupByNode(key)`,
		`datamodel.KindSet_Recursive`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodLookupBySegment(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`LookupBySegment`,
		`LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error)`,
		`{{- if or (.Type.RepresentationStrategy.GetMember (Kind "map")) (.Type.RepresentationStrategy.GetMember (Kind "list")) }}`,
		`{{- if or (eq $member.RepresentationBehavior.String "map") (eq $member.RepresentationBehavior.String "list") }}`,
		`.LookupBySegment(seg)`,
		`datamodel.KindSet_Recursive`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`MapIterator`,
		`MapIterator() datamodel.MapIterator`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "map") }}`,
		`{{- if eq $member.RepresentationBehavior.String "map" }}`,
		`.MapIterator()`,
		`datamodel.KindSet_JustMap`,
		`nil`,
		true,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodListIterator(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`ListIterator`,
		`ListIterator() datamodel.ListIterator`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "list") }}`,
		`{{- if eq $member.RepresentationBehavior.String "list" }}`,
		`.ListIterator()`,
		`datamodel.KindSet_JustList`,
		`nil`,
		true,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`Length`,
		`Length() int64`,
		`{{- if or (.Type.RepresentationStrategy.GetMember (Kind "map")) (.Type.RepresentationStrategy.GetMember (Kind "list")) }}`,
		`{{- if or (eq $member.RepresentationBehavior.String "map") (eq $member.RepresentationBehavior.String "list") }}`,
		`.Length()`,
		`datamodel.KindSet_Recursive`,
		`-1`,
		true,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodIsAbsent(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) IsAbsent() bool {
			return false
		}
	`, w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodIsNull(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) IsNull() bool {
			return false
		}
	`, w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsBool(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsBool`,
		`AsBool() (bool, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "bool") }}`,
		`{{- if eq $member.RepresentationBehavior.String "bool" }}`,
		`.AsBool()`,
		`datamodel.KindSet_JustBool`,
		`false`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsInt(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsInt`,
		`AsInt() (int64, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "int") }}`,
		`{{- if eq $member.RepresentationBehavior.String "int" }}`,
		`.AsInt()`,
		`datamodel.KindSet_JustInt`,
		`0`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsFloat(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsFloat`,
		`AsFloat() (float64, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "float") }}`,
		`{{- if eq $member.RepresentationBehavior.String "float" }}`,
		`.AsFloat()`,
		`datamodel.KindSet_JustFloat`,
		`0`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsString(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsString`,
		`AsString() (string, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "string") }}`,
		`{{- if eq $member.RepresentationBehavior.String "string" }}`,
		`.AsString()`,
		`datamodel.KindSet_JustString`,
		`""`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsBytes(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsBytes`,
		`AsBytes() ([]byte, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "bytes") }}`,
		`{{- if eq $member.RepresentationBehavior.String "bytes" }}`,
		`.AsBytes()`,
		`datamodel.KindSet_JustBytes`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodAsLink(w io.Writer) {
	doTemplate(kindedUnionNodeMethodTemplateMunge(
		`AsLink`,
		`AsLink() (datamodel.Link, error)`,
		`{{- if .Type.RepresentationStrategy.GetMember (Kind "link") }}`,
		`{{- if eq $member.RepresentationBehavior.String "link" }}`,
		`.AsLink()`,
		`datamodel.KindSet_JustLink`,
		`nil`,
		false,
	), w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g unionKindedReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g unionKindedReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return unionKindedReprBuilderGenerator(g)
}

type unionKindedReprBuilderGenerator struct {
	AdjCfg  *AdjunctCfg
	PkgName string
	Type    *schema.TypeUnion
}

func (unionKindedReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionKindedReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Much of this is familiar: the 'w', the 'm' are all as usual.
	// Some things may look a little odd here compared to all other assemblers:
	//  we're kinda halfway between what's conventionally seen for a scalar and what's conventionally seen for a recursive.
	// There's no 'maState' or 'laState'-typed fields (which feels like a scalar) because even if we end up acting like a map or list, that state is in the relevant child assembler.
	// We don't even have a 'cm' field, because we can get away with something really funky: we can just copy our own 'm' _pointer_ into children; our doneness and their doneness is the same.
	// We never have to worry about maybeism of our children; the nullable and optional modifiers aren't possible on union members.
	//  (We *do* still have to consider null values though, as null is still a kind, and thus can be routed to one of our members!)
	// 'ca' is as it is in the type-level assembler: technically, not super necessary, except that it allows minimizing the amount of work that resetting needs to do.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe

			{{- range $i, $member := .Type.Members }}
			ca{{ add $i 1 }} {{ if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}*{{end}}_{{ $member | TypeSymbol }}__ReprAssembler
			{{- end}}
			ca uint
		}
	`, w, g.AdjCfg, g)
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			switch na.ca {
			case 0:
				return
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				na.ca{{ add $i 1 }}.reset()
			{{- end}}
			default:
				panic("unreachable")
			}
			na.ca = 0
		}
	`, w, g.AdjCfg, g)
}

func kindedUnionNodeAssemblerMethodTemplateMunge(
	methodName string, // for error messages
	methodSig string, // output literally
	condClause string, // template condition for the member this should match on
	retClause string, // clause returning the thing (how to delegate methodsig, generally)
	twoReturns bool, // true if a nil should be returned as well as the error
) string {
	// The value pointed to by `na.m` isn't modified here, because we're sharing it with the child, who should do so.
	//  This also means that value gets checked twice -- once by us, because we need to halt if we've already been used --
	//   and also a second time by the child when we delegate to it, which, unbeknownst to it, is irrelevant.
	//   I don't see a good way to remedy this shy of making more granular (unexported!) methods.  (Might be worth it.)
	//   This probably also isn't the same for all of the assembler methods: the methods we delegate to aren't doing as many check branches when they're for scalars,
	//    because they expected to be used in contexts where many values of the 'm' enum aren't reachable -- an expectation we've suddenly subverted with this path!
	//
	// FUTURE: The error returns here are deeply questionable, and not as helpful as they could be.
	//  ErrNotUnionStructure is about the most applicable thing so far, but it's very freetext.
	//  ErrWrongKind wouldn't fit at all: assumes that we can say what kind of node we have, but this is the one case in the whole system where *we can't*; also, assumes there's one actual correct kind, and that too is false here!
	maybeNilComma := ""
	if twoReturns {
		maybeNilComma += "nil,"
	}
	return `
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) ` + methodSig + ` {
			switch *na.m {
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			case midvalue:
				panic("invalid state: cannot assign into assembler that's already working on a larger structure!")
			}
			{{- $returned := false -}}
			{{- range $i, $member := .Type.Members }}
			` + condClause + `
				{{- if dot.Type | MaybeUsesPtr }}
					if na.w == nil {
						na.w = &_{{ dot.Type | TypeSymbol }}{}
					}
				{{- end}}
				na.ca = {{ add $i 1 }}
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
					na.w.tag = {{ add $i 1 }}
					na.ca{{ add $i 1 }}.w = &na.w.x{{ add $i 1 }}
					na.ca{{ add $i 1 }}.m = na.m
					return na.ca{{ add $i 1 }}` + retClause + `
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
					x := &_{{ $member | TypeSymbol }}{}
					na.w.x = x
					if na.ca{{ add $i 1 }} == nil {
						na.ca{{ add $i 1 }} = &_{{ $member | TypeSymbol }}__ReprAssembler{}
					}
					na.ca{{ add $i 1 }}.w = x
					na.ca{{ add $i 1 }}.m = na.m
					return na.ca{{ add $i 1 }}` + retClause + `
				{{- end}}
				{{- $returned = true -}}
			{{- end }}
			{{- end }}
			{{- if not $returned }}
			return ` + maybeNilComma + ` schema.ErrNotUnionStructure{TypeName: "{{ .PkgName }}.{{ .Type.Name }}.Repr", Detail: "` + methodName + ` called but is not valid for any of the kinds that are valid members of this union"}
			{{- end }}
		}
	`
}

func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`BeginMap`,
		`BeginMap(sizeHint int64) (datamodel.MapAssembler, error)`,
		`{{- if eq $member.RepresentationBehavior.String "map" }}`,
		`.BeginMap(sizeHint)`,
		true,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`BeginList`,
		`BeginList(sizeHint int64) (datamodel.ListAssembler, error)`,
		`{{- if eq $member.RepresentationBehavior.String "list" }}`,
		`.BeginList(sizeHint)`,
		true,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	// TODO: I think this may need some special handling to account for if our union is itself used in a nullable circumstance; that should overrule this behavior.
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignNull`,
		`AssignNull() error `,
		`{{- if eq $member.RepresentationBehavior.String "null" }}`,
		`.AssignNull()`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignBool`,
		`AssignBool(v bool) error `,
		`{{- if eq $member.RepresentationBehavior.String "bool" }}`,
		`.AssignBool(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignInt`,
		`AssignInt(v int64) error `,
		`{{- if eq $member.RepresentationBehavior.String "int" }}`,
		`.AssignInt(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignFloat`,
		`AssignFloat(v float64) error `,
		`{{- if eq $member.RepresentationBehavior.String "float" }}`,
		`.AssignFloat(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignString`,
		`AssignString(v string) error `,
		`{{- if eq $member.RepresentationBehavior.String "string" }}`,
		`.AssignString(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignBytes`,
		`AssignBytes(v []byte) error `,
		`{{- if eq $member.RepresentationBehavior.String "bytes" }}`,
		`.AssignBytes(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(w io.Writer) {
	doTemplate(kindedUnionNodeAssemblerMethodTemplateMunge(
		`AssignLink`,
		`AssignLink(v datamodel.Link) error `,
		`{{- if eq $member.RepresentationBehavior.String "link" }}`,
		`.AssignLink(v)`,
		false,
	), w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// This is a very mundane AssignNode: it just calls out to the other methods on this type.
	//  However, even that is a little more exciting than usual: because we can't *necessarily* reject any kind of arg,
	//   we have the whole barrage of switch cases here.  We then leave any particular rejections to those methods.
	//  Several cases could be statically replaced with errors and it would be an improvement.
	//
	// Errors are problematic again, same as is noted in kindedUnionNodeAssemblerMethodTemplateMunge.
	//  We also end up returning errors with other method names due to how we delegate; unfortunate.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) AssignNode(v datamodel.Node) error {
			if v.IsNull() {
				return na.AssignNull()
			}
			if v2, ok := v.(*_{{ .Type | TypeSymbol }}); ok {
				switch *na.m {
				case schema.Maybe_Value, schema.Maybe_Null:
					panic("invalid state: cannot assign into assembler that's already finished")
				case midvalue:
					panic("invalid state: cannot assign null into an assembler that's already begun working on recursive structures!")
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
			switch v.Kind() {
			case datamodel.Kind_Bool:
				v2, _ := v.AsBool()
				return na.AssignBool(v2)
			case datamodel.Kind_Int:
				v2, _ := v.AsInt()
				return na.AssignInt(v2)
			case datamodel.Kind_Float:
				v2, _ := v.AsFloat()
				return na.AssignFloat(v2)
			case datamodel.Kind_String:
				v2, _ := v.AsString()
				return na.AssignString(v2)
			case datamodel.Kind_Bytes:
				v2, _ := v.AsBytes()
				return na.AssignBytes(v2)
			case datamodel.Kind_Map:
				na, err := na.BeginMap(v.Length())
				if err != nil {
					return err
				}
				itr := v.MapIterator()
				for !itr.Done() {
					k, v, err := itr.Next()
					if err != nil {
						return err
					}
					if err := na.AssembleKey().AssignNode(k); err != nil {
						return err
					}
					if err := na.AssembleValue().AssignNode(v); err != nil {
						return err
					}
				}
				return na.Finish()
			case datamodel.Kind_List:
				na, err := na.BeginList(v.Length())
				if err != nil {
					return err
				}
				itr := v.ListIterator()
				for !itr.Done() {
					_, v, err := itr.Next()
					if err != nil {
						return err
					}
					if err := na.AssembleValue().AssignNode(v); err != nil {
						return err
					}
				}
				return na.Finish()
			case datamodel.Kind_Link:
				v2, _ := v.AsLink()
				return na.AssignLink(v2)
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(w io.Writer) {
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) Prototype() datamodel.NodePrototype {
			return _{{ .Type | TypeSymbol }}__ReprPrototype{}
		}
	`, w, g.AdjCfg, g)
}
func (g unionKindedReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// somewhat shockingly: nothing.
}
