package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type structGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g structGenerator) EmitNativeType(w io.Writer) {
	doTemplate(`
		{{- if Comments -}}
		// {{ .Type | TypeSymbol }} matches the IPLD Schema type "{{ .Type.Name }}".  It has {{ .Type.TypeKind }} type-kind, and may be interrogated like {{ .Kind }} kind.
		{{- end}}
		type {{ .Type | TypeSymbol }} = *_{{ .Type | TypeSymbol }}
		type _{{ .Type | TypeSymbol }} struct {
			{{- range $field := .Type.Fields}}
			{{ $field | FieldSymbolLower }} _{{ $field.Type | TypeSymbol }}{{if $field.IsMaybe }}__Maybe{{end}}
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNativeAccessors(w io.Writer) {
	doTemplate(`
		{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
		{{- range $field := .Type.Fields }}
		func (n _{{ $type | TypeSymbol }}) Field{{ $field | FieldSymbolUpper }}() {{ if $field.IsMaybe }}Maybe{{end}}{{ $field.Type | TypeSymbol }} {
			return &n.{{ $field | FieldSymbolLower }}
		}
		{{- end}}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNativeBuilder(w io.Writer) {
	// Unclear what, if anything, goes here.
}

func (g structGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g structGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g structGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g structGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
	// We do, however, want some constants for our fields;
	//  they'll make iterators able to work faster.  So let's emit those.
	doTemplate(`
		var (
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $field := .Type.Fields }}
			fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} = _String{"{{ $field.Name }}"}
			{{- end }}
		)
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByString(key string) (datamodel.Node, error) {
			switch key {
			{{- range $field := .Type.Fields }}
			case "{{ $field.Name }}":
				{{- if $field.IsOptional }}
				if n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					return datamodel.Absent, nil
				}
				{{- end}}
				{{- if $field.IsNullable }}
				if n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Null {
					return datamodel.Null, nil
				}
				{{- end}}
				{{- if $field.IsMaybe }}
				return {{if not (MaybeUsesPtr $field.Type) }}&{{end}}n.{{ $field | FieldSymbolLower }}.v, nil
				{{- else}}
				return &n.{{ $field | FieldSymbolLower }}, nil
				{{- end}}
			{{- end}}
			default:
				return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfString(key)}
			}
		}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
			ks, err := key.AsString()
			if err != nil {
				return nil, err
			}
			return n.LookupByString(ks)
		}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	// Note that the typed iterator will report absent fields.
	//  The representation iterator (if has one) however will skip those.
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) MapIterator() datamodel.MapIterator {
			return &_{{ .Type | TypeSymbol }}__MapItr{n, 0}
		}

		type _{{ .Type | TypeSymbol }}__MapItr struct {
			n {{ .Type | TypeSymbol }}
			idx  int
		}

		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Next() (k datamodel.Node, v datamodel.Node, _ error) {
			{{- if not .Type.Fields }}
			return nil, nil, datamodel.ErrIteratorOverread{}
			{{ else -}}
			if itr.idx >= {{ len .Type.Fields }} {
				return nil, nil, datamodel.ErrIteratorOverread{}
			}
			switch itr.idx {
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				k = &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}
				{{- if $field.IsOptional }}
				if itr.n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					v = datamodel.Absent
					break
				}
				{{- end}}
				{{- if $field.IsNullable }}
				if itr.n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Null {
					v = datamodel.Null
					break
				}
				{{- end}}
				{{- if $field.IsMaybe }}
				v = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}itr.n.{{ $field | FieldSymbolLower}}.v
				{{- else}}
				v = &itr.n.{{ $field | FieldSymbolLower}}
				{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
			itr.idx++
			return
			{{- end}}
		}
		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Done() bool {
			return itr.idx >= {{ len .Type.Fields }}
		}

	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Length() int64 {
			return {{ len .Type.Fields }}
		}
	`, w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g structGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g structGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return structBuilderGenerator{
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

type structBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g structBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g structBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g structBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the **m**aybe which communicates our completeness to the parent if we're a child assembler.
	// - 'state' is what it says on the tin.  this is used for the map state (the broad transitions between null, start-map, and finish are handled by 'm' for consistency.)
	// - 's' is a bitfield for what's been **s**et.
	// - 'f' is the **f**ocused field that will be assembled next.
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children that aren't allowed to be nullable (for those that are, their own maybe.m is used).
	//    ('cm' could be elided for structs where all fields are maybes.  trivial but not yet implemented.)
	// - the 'ca_*' fields embed **c**hild **a**ssemblers -- these are embedded so we can yield pointers to them without causing new allocations.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Assembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState
			s int
			f int

			cm schema.Maybe
			{{range $field := .Type.Fields -}}
			ca_{{ $field | FieldSymbolLower }} _{{ $field.Type | TypeSymbol }}__Assembler
			{{end -}}
		}

		func (na *_{{ .Type | TypeSymbol }}__Assembler) reset() {
			na.state = maState_initial
			na.s = 0
			{{- range $field := .Type.Fields }}
			na.ca_{{ $field | FieldSymbolLower }}.reset()
			{{- end}}
		}

		var (
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} = 1 << {{ $i }}
			{{- end}}
			fieldBits__{{ $type | TypeSymbol }}_sufficient = 0 {{- range $i, $field := .Type.Fields }}{{if not $field.IsOptional }} + 1 << {{ $i }}{{end}}{{end}}
		)
	`, w, g.AdjCfg, g)
}
func (g structBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_strictoid(w, g.AdjCfg, g)
}
func (g structBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g structBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	//
	// We do not set m=midvalue in phase 3 -- it shouldn't matter unless you're trying to pull off concurrent access, which is wrong and unsafe regardless.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__Assembler) AssignNode(v datamodel.Node) error {
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
				return datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustMap, ActualKind: v.Kind()}
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
func (g structBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	g.emitMapAssemblerChildTidyHelper(w)
	g.emitMapAssemblerMethods(w)
	g.emitKeyAssembler(w)
}
func (g structBuilderGenerator) emitMapAssemblerChildTidyHelper(w io.Writer) {
	// This function attempts to clean up the state machine to acknolwedge child assembly finish.
	//  If the child was finished and we just collected it, return true and update state to maState_initial.
	//  Otherwise, if it wasn't done, return false;
	//   and the caller is almost certain to emit an error momentarily.
	// The function will only be called when the current state is maState_midValue.
	//  (In general, the idea is that if the user is doing things correctly,
	//   this function will only be called when the child is in fact finished.)
	// Most of the logic here is about nullables and not optionals,
	//  because if you're an optional that's absent, you never got to value assembly.
	//  There's still one branch for optionals, though, because they have a different residence for 'm' just as nullables do.
	// Child assemblers are expected to control their own state machines;
	//  for values that have maybes, we never change their maybe state again, so the usual logic should hold;
	//  for values that don't have maybes (and thus share 'cm')...
	//   We don't bother to nil their 'm' pointer; the worst that can happen is an over-held assembler for that field
	//    can make a bizarre and broken transition for a subsequent field, which will result in very ugly errors, but isn't unsafe per se.
	//   We do nil their 'w' pointer, though: we don't want a set to that able to leak in later if we're on the way to Finish!
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) valueFinishTidy() bool {
			switch ma.f {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsNullable }}
				switch ma.w.{{ $field | FieldSymbolLower }}.m {
				case schema.Maybe_Null:
					ma.state = maState_initial
					return true
				case schema.Maybe_Value:
					{{- if (MaybeUsesPtr $field.Type) }}
					ma.w.{{ $field | FieldSymbolLower }}.v = ma.ca_{{ $field | FieldSymbolLower }}.w
					{{- end}}
					ma.state = maState_initial
					return true
				default:
					return false
				}
				{{- else if $field.IsOptional }}
				switch ma.w.{{ $field | FieldSymbolLower }}.m {
				case schema.Maybe_Value:
					{{- if (MaybeUsesPtr $field.Type) }}
					ma.w.{{ $field | FieldSymbolLower }}.v = ma.ca_{{ $field | FieldSymbolLower }}.w
					{{- end}}
					ma.state = maState_initial
					return true
				default:
					return false
				}
				{{- else}}
				switch ma.cm {
				case schema.Maybe_Value:
					ma.ca_{{ $field | FieldSymbolLower }}.w = nil
					ma.cm = schema.Maybe_Absent
					ma.state = maState_initial
					return true
				default:
					return false
				}
				{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)
}
func (g structBuilderGenerator) emitMapAssemblerMethods(w io.Writer) {
	// FUTURE: some of the setup of the child assemblers could probably be DRY'd up.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
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
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: AssembleEntry cannot be called on an assembler that's already finished")
			}
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- if .Type.Fields }}
			switch k {
			{{- range $i, $field := .Type.Fields }}
			case "{{ $field.Name }}":
				if ma.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} != 0 {
					return nil, datamodel.ErrRepeatedMapKey{Key: &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}}
				}
				ma.s += fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}
				ma.state = maState_midValue
				ma.f = {{ $i }}
				{{- if $field.IsMaybe }}
				ma.ca_{{ $field | FieldSymbolLower }}.w = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}ma.w.{{ $field | FieldSymbolLower }}.v
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.w.{{ $field | FieldSymbolLower }}.m
				{{- if $field.IsNullable }}
				ma.w.{{ $field | FieldSymbolLower }}.m = allowNull
				{{- end}}
				{{- else}}
				ma.ca_{{ $field | FieldSymbolLower }}.w = &ma.w.{{ $field | FieldSymbolLower }}
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.cm
				{{- end}}
				return &ma.ca_{{ $field | FieldSymbolLower }}, nil
			{{- end}}
			}
			{{- end}}
			return nil, schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Key:&_String{k}}
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleKey() datamodel.NodeAssembler {
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
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: AssembleKey cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midKey
			return (*_{{ .Type | TypeSymbol }}__KeyAssembler)(ma)
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleValue() datamodel.NodeAssembler {
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
			switch ma.f {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsMaybe }}
				ma.ca_{{ $field | FieldSymbolLower }}.w = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}ma.w.{{ $field | FieldSymbolLower }}.v
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.w.{{ $field | FieldSymbolLower }}.m
				{{- if $field.IsNullable }}
				ma.w.{{ $field | FieldSymbolLower }}.m = allowNull
				{{- end}}
				{{- else}}
				ma.ca_{{ $field | FieldSymbolLower }}.w = &ma.w.{{ $field | FieldSymbolLower }}
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.cm
				{{- end}}
				return &ma.ca_{{ $field | FieldSymbolLower }}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) Finish() error {
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
			if ma.s & fieldBits__{{ $type | TypeSymbol }}_sufficient != fieldBits__{{ $type | TypeSymbol }}_sufficient {
				err := schema.ErrMissingRequiredField{Missing: make([]string, 0)}
				{{- range $i, $field := .Type.Fields }}
				{{- if not $field.IsMaybe}}
				if ma.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} == 0 {
					err.Missing = append(err.Missing, "{{ $field.Name }}")
				}
				{{- end}}
				{{- end}}
				return err
			}
			ma.state = maState_finished
			*ma.m = schema.Maybe_Value
			return nil
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) KeyPrototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) ValuePrototype(k string) datamodel.NodePrototype {
			panic("todo structbuilder mapassembler valueprototype")
		}
	`, w, g.AdjCfg, g)
}
func (g structBuilderGenerator) emitKeyAssembler(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__KeyAssembler _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
	stubs := mixins.StringAssemblerTraits{
		PkgName:       g.PkgName,
		TypeName:      g.TypeName + ".KeyAssembler",
		AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Key",
	}
	// This key assembler can disregard any idea of complex keys because it's a struct!
	//  Struct field names must be strings (and quite simple ones at that).
	stubs.EmitNodeAssemblerMethodBeginMap(w)
	stubs.EmitNodeAssemblerMethodBeginList(w)
	stubs.EmitNodeAssemblerMethodAssignNull(w)
	stubs.EmitNodeAssemblerMethodAssignBool(w)
	stubs.EmitNodeAssemblerMethodAssignInt(w)
	stubs.EmitNodeAssemblerMethodAssignFloat(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__KeyAssembler) AssignString(k string) error {
			if ka.state != maState_midKey {
				panic("misuse: KeyAssembler held beyond its valid lifetime")
			}
			switch k {
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			case "{{ $field.Name }}":
				if ka.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} != 0 {
					return datamodel.ErrRepeatedMapKey{Key: &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}}
				}
				ka.s += fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}
				ka.state = maState_expectValue
				ka.f = {{ $i }}
				return nil
			{{- end}}
			default:
				return schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Key:&_String{k}}
			}
		}
	`, w, g.AdjCfg, g)
	stubs.EmitNodeAssemblerMethodAssignBytes(w)
	stubs.EmitNodeAssemblerMethodAssignLink(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__KeyAssembler) AssignNode(v datamodel.Node) error {
			if v2, err := v.AsString(); err != nil {
				return err
			} else {
				return ka.AssignString(v2)
			}
		}
		func (_{{ .Type | TypeSymbol }}__KeyAssembler) Prototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
	`, w, g.AdjCfg, g)
}
