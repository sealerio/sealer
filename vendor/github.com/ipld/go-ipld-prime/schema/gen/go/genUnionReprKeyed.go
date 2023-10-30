package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &unionReprKeyedGenerator{}

// General observation: many things about the keyed representation of unions is *very* similar to the type-level code,
//  because the type level code effective does espouse keyed-style behavior (just with type names as the keys).
//  Be advised that this similarity does not hold at *all* true of any of the other representation modes of unions!

func NewUnionReprKeyedGenerator(pkgName string, typ *schema.TypeUnion, adjCfg *AdjunctCfg) TypeGenerator {
	return unionReprKeyedGenerator{
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

type unionReprKeyedGenerator struct {
	unionGenerator
}

func (g unionReprKeyedGenerator) GetRepresentationNodeGen() NodeGenerator {
	return unionReprKeyedReprGenerator{
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

type unionReprKeyedReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionReprKeyedReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionReprKeyedReprGenerator) EmitNodeType(w io.Writer) {
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

func (g unionReprKeyedReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	// Similar to the type-level method, except uses discriminant values as keys instead of the member type names.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) LookupByString(key string) (datamodel.Node, error) {
			switch key {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member | dot.Type.RepresentationStrategy.GetDiscriminant }}":
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				if n.tag != {{ add $i 1 }} {
					return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
				}
				return n.x{{ add $i 1 }}.Representation(), nil
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				if n2, ok := n.x.({{ $member | TypeSymbol }}); ok {
					return n2.Representation(), nil
				} else {
					return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
				}
				{{- end}}
			{{- end}}
			default:
				return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfString(key)}
			}
		}
	`, w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
			ks, err := key.AsString()
			if err != nil {
				return nil, err
			}
			return n.LookupByString(ks)
		}
	`, w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	// Similar to the type-level method, except yields discriminant values as keys instead of the member type names.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) MapIterator() datamodel.MapIterator {
			return &_{{ .Type | TypeSymbol }}__ReprMapItr{n, false}
		}

		type _{{ .Type | TypeSymbol }}__ReprMapItr struct {
			n *_{{ .Type | TypeSymbol }}__Repr
			done bool
		}

		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Next() (k datamodel.Node, v datamodel.Node, _ error) {
			if itr.done {
				return nil, nil, datamodel.ErrIteratorOverread{}
			}
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch itr.n.tag {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				k, v = &memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}_serial, itr.n.x{{ add $i 1 }}.Representation()
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			switch n2 := itr.n.x.(type) {
			{{- range $member := .Type.Members }}
			case {{ $member | TypeSymbol }}:
				k, v = &memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}_serial, n2.Representation()
			{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
			itr.done = true
			return
		}
		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Done() bool {
			return itr.done
		}

	`, w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func (_{{ .Type | TypeSymbol }}__Repr) Length() int64 {
			return 1
		}
	`, w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g unionReprKeyedReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g unionReprKeyedReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return unionReprKeyedReprBuilderGenerator{
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

type unionReprKeyedReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionReprKeyedReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g unionReprKeyedReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Nearly identical to the type-level system, except it embeds the Repr variant of child assemblers
	//  (which is a very minor difference textually, but means this structure can end up with a pretty wildly different resident memory size than the type-level one).
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState

			cm schema.Maybe
			{{- range $i, $member := .Type.Members }}
			ca{{ add $i 1 }} {{ if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}*{{end}}_{{ $member | TypeSymbol }}__ReprAssembler
			{{end -}}
			ca uint
		}
	`, w, g.AdjCfg, g)

	// Reset methods: also nearly identical to the type-level ones.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			na.state = maState_initial
			switch na.ca {
			case 0:
				return
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				na.ca{{ add $i 1 }}.reset()
			{{end -}}
			default:
				panic("unreachable")
			}
			na.ca = 0
			na.cm = schema.Maybe_Absent
		}
	`, w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_strictoid(w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	// It might sound a bit odd to call a union "recursive", since it's so very trivially so (no fan-out),
	//  but it's functionally accurate: the generated method should include a branch for the 'midvalue' state.
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// DRY: this is once again not-coincidentally very nearly equal to the type-level method.  Would be good to dedup them... after we do the get-to-the-point-in-phase-3 improvement.
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
			if v.Kind() != datamodel.Kind_Map {
				return datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}.Repr", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustMap, ActualKind: v.Kind()}
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
		}
	`, w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	g.emitMapAssemblerChildTidyHelper(w)
	g.emitMapAssemblerMethods(w)
	g.emitKeyAssembler(w)
}
func (g unionReprKeyedReprBuilderGenerator) emitMapAssemblerChildTidyHelper(w io.Writer) {
	// Nearly identical to the type-level equivalent.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) valueFinishTidy() bool {
			switch ma.cm {
			case schema.Maybe_Value:
				{{- /* nothing to do for memlayout=embedAll; the tag is already set and memory already in place. */ -}}
				{{- /* nothing to do for memlayout=interface either; same story, the values are already in place. */ -}}
				ma.state = maState_initial
				return true
			default:
				return false
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) emitMapAssemblerMethods(w io.Writer) {
	// All of these: shamelessly similar to the type-level equivalent, modulo a few appearances of "Repr".
	//  Alright, and also the "discriminant values as keys instead of the member type names" thing.
	// DRY: the number of times these `ma.state` switches are appearing is truly intense!  This is starting to look like one of them most important things to shrink the GSLOC/ASM size of!

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: AssembleEntry cannot be called when in the middle of assembling another key")
			case maState_expectValue:
				panic("invalid state: AssembleEntry cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: AssembleEntry cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on for the moment, but we'll still be erroring shortly.
			case maState_finished:
				panic("invalid state: AssembleEntry cannot be called on an assembler that's already finished")
			}
			if ma.ca != 0 {
				return nil, schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Detail: "cannot add another entry -- a union can only contain one thing!"}
			}
			{{- if .Type.Members }}
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member | dot.Type.RepresentationStrategy.GetDiscriminant }}":
				ma.state = maState_midValue
				ma.ca = {{ add $i 1 }}
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ma.w.tag = {{ add $i 1 }}
				ma.ca{{ add $i 1 }}.w = &ma.w.x{{ add $i 1 }}
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return &ma.ca{{ add $i 1 }}, nil
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				x := &_{{ $member | TypeSymbol }}{}
				ma.w.x = x
				if ma.ca{{ add $i 1 }} == nil {
					ma.ca{{ add $i 1 }} = &_{{ $member | TypeSymbol }}__ReprAssembler{}
				}
				ma.ca{{ add $i 1 }}.w = x
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return ma.ca{{ add $i 1 }}, nil
				{{- end}}
			{{- end}}
			}
			{{- end}}
			return nil, schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Key:&_String{k}}
		}
	`, w, g.AdjCfg, g)

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) AssembleKey() datamodel.NodeAssembler {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: AssembleKey cannot be called when in the middle of assembling another key")
			case maState_expectValue:
				panic("invalid state: AssembleKey cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: AssembleKey cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on for the moment, but we'll still be erroring shortly... or rather, the keyassembler will be.
			case maState_finished:
				panic("invalid state: AssembleKey cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midKey
			return (*_{{ .Type | TypeSymbol }}__ReprKeyAssembler)(ma)
		}
	`, w, g.AdjCfg, g)

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) AssembleValue() datamodel.NodeAssembler {
			switch ma.state {
			case maState_initial:
				panic("invalid state: AssembleValue cannot be called when no key is primed")
			case maState_midKey:
				panic("invalid state: AssembleValue cannot be called when in the middle of assembling a key")
			case maState_expectValue:
				// carry on
			case maState_midValue:
				panic("invalid state: AssembleValue cannot be called when in the middle of assembling another value")
			case maState_finished:
				panic("invalid state: AssembleValue cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midValue
			switch ma.ca {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ma.ca{{ add $i 1 }}.w = &ma.w.x{{ add $i 1 }}
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return &ma.ca{{ add $i 1 }}
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				x := &_{{ $member | TypeSymbol }}{}
				ma.w.x = x
				if ma.ca{{ add $i 1 }} == nil {
					ma.ca{{ add $i 1 }} = &_{{ $member | TypeSymbol }}__ReprAssembler{}
				}
				ma.ca{{ add $i 1 }}.w = x
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return ma.ca{{ add $i 1 }}
				{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) Finish() error {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: Finish cannot be called when in the middle of assembling a key")
			case maState_expectValue:
				panic("invalid state: Finish cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: Finish cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: Finish cannot be called on an assembler that's already finished")
			}
			if ma.ca == 0 {
				return schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Detail: "a union must have exactly one entry (not none)!"}
			}
			ma.state = maState_finished
			*ma.m = schema.Maybe_Value
			return nil
		}
	`, w, g.AdjCfg, g)

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) KeyPrototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) ValuePrototype(k string) datamodel.NodePrototype {
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member.Name }}":
				return _{{ $member | TypeSymbol }}__ReprPrototype{}
			{{- end}}
			default:
				return nil
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionReprKeyedReprBuilderGenerator) emitKeyAssembler(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprKeyAssembler _{{ .Type | TypeSymbol }}__ReprAssembler
	`, w, g.AdjCfg, g)
	stubs := mixins.StringAssemblerTraits{
		PkgName:       g.PkgName,
		TypeName:      g.TypeName + ".KeyAssembler", // ".Repr" is already in `g.TypeName`, so don't stutter the "Repr" part.
		AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__ReprKey",
	}
	// This key assembler can disregard any idea of complex keys because we know that our discriminants are just strings!
	stubs.EmitNodeAssemblerMethodBeginMap(w)
	stubs.EmitNodeAssemblerMethodBeginList(w)
	stubs.EmitNodeAssemblerMethodAssignNull(w)
	stubs.EmitNodeAssemblerMethodAssignBool(w)
	stubs.EmitNodeAssemblerMethodAssignInt(w)
	stubs.EmitNodeAssemblerMethodAssignFloat(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__ReprKeyAssembler) AssignString(k string) error {
			if ka.state != maState_midKey {
				panic("misuse: KeyAssembler held beyond its valid lifetime")
			}
			if ka.ca != 0 {
				return schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Detail: "cannot add another entry -- a union can only contain one thing!"}
			}
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member | dot.Type.RepresentationStrategy.GetDiscriminant }}":
				ka.ca = {{ add $i 1 }}
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ka.w.tag = {{ add $i 1 }}
				{{- end}}
				ka.state = maState_expectValue
				return nil
			{{- end}}
			}
			return schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Key:&_String{k}} // TODO: error quality: ErrInvalidUnionDiscriminant ?
		}
	`, w, g.AdjCfg, g)
	stubs.EmitNodeAssemblerMethodAssignBytes(w)
	stubs.EmitNodeAssemblerMethodAssignLink(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__ReprKeyAssembler) AssignNode(v datamodel.Node) error {
			if v2, err := v.AsString(); err != nil {
				return err
			} else {
				return ka.AssignString(v2)
			}
		}
		func (_{{ .Type | TypeSymbol }}__ReprKeyAssembler) Prototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
	`, w, g.AdjCfg, g)
}
