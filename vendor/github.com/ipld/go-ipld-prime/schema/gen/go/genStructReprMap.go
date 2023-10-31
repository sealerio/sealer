package gengo

import (
	"io"
	"strconv"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &structReprMapGenerator{}

func NewStructReprMapGenerator(pkgName string, typ *schema.TypeStruct, adjCfg *AdjunctCfg) TypeGenerator {
	return structReprMapGenerator{
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

type structReprMapGenerator struct {
	structGenerator
}

func (g structReprMapGenerator) GetRepresentationNodeGen() NodeGenerator {
	return structReprMapReprGenerator{
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

type structReprMapReprGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprMapReprGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprMapReprGenerator) EmitNodeType(w io.Writer) {
	// The type is structurally the same, but will have a different set of methods.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)

	// We do also want some constants for our fields;
	//  they'll make iterators able to work faster.
	//  These might be the same strings as the type-level field names
	//   (in fact, they are, unless renames are used)... but that's fine.
	//    We get simpler code by just doing this unconditionally.
	doTemplate(`
		var (
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $field := .Type.Fields }}
			fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}_serial = _String{"{{ $field | $type.RepresentationStrategy.GetFieldKey }}"}
			{{- end }}
		)
	`, w, g.AdjCfg, g)
}

func (g structReprMapReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}

func (g structReprMapReprGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	// Similar to the type-level method, except any absent fields also return ErrNotExists.
	doTemplate(`
		func (n *_{{ .Type | TypeSymbol }}__Repr) LookupByString(key string) (datamodel.Node, error) {
			switch key {
			{{- range $field := .Type.Fields }}
			case "{{ $field | $field.Parent.RepresentationStrategy.GetFieldKey }}":
				{{- if $field.IsOptional }}
				if n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					return datamodel.Absent, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
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
				return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfString(key)}
			}
		}
	`, w, g.AdjCfg, g)
}

func (g structReprMapReprGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
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

func (g structReprMapReprGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	// The 'idx' int is what field we'll yield next.
	// Note that this iterator doesn't mention fields that are absent.
	//  This makes things a bit trickier -- especially the 'Done' predicate,
	//   since it may have to do lookahead if there's any optionals at the end of the structure!
	//  It also means 'idx' can jump ahead by more than one per Next call in order to skip over absent fields.
	// TODO : support for implicits is still future work.

	// First: Determine if there are any optionals at all.
	//  If there are none, some control flow symbols need to not be emitted.
	fields := g.Type.Fields()
	haveOptionals := false
	for _, field := range fields {
		if field.IsOptional() {
			haveOptionals = true
			break
		}
	}

	// Second: Count how many trailing fields are optional.
	//  The 'Done' predicate gets more complex when in the trailing optionals.
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
		func (n *_{{ .Type | TypeSymbol }}__Repr) MapIterator() datamodel.MapIterator {
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
			return &_{{ .Type | TypeSymbol }}__ReprMapItr{n, 0, end}
			{{- else}}
			return &_{{ .Type | TypeSymbol }}__ReprMapItr{n, 0}
			{{- end}}
		}

		type _{{ .Type | TypeSymbol }}__ReprMapItr struct {
			n   *_{{ .Type | TypeSymbol }}__Repr
			idx int
			{{if .HaveTrailingOptionals }}end int{{end}}
		}

		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Next() (k datamodel.Node, v datamodel.Node, _ error) {
			{{- if not .Type.Fields }}
			{{- /* TODO: deduplicate all these methods which just error */ -}}
			return nil, nil, datamodel.ErrIteratorOverread{}
			{{ else -}}
			{{ if .HaveOptionals }}advance:{{end -}}
			if itr.idx >= {{ len .Type.Fields }} {
				return nil, nil, datamodel.ErrIteratorOverread{}
			}
			switch itr.idx {
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				k = &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}_serial
				{{- if $field.IsOptional }}
				if itr.n.{{ $field | FieldSymbolLower }}.m == schema.Maybe_Absent {
					itr.idx++
					goto advance
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
			{{- end}}
		}
		{{- if .HaveTrailingOptionals }}
		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Done() bool {
			return itr.idx >= itr.end
		}
		{{- else}}
		func (itr *_{{ .Type | TypeSymbol }}__ReprMapItr) Done() bool {
			return itr.idx >= {{ len .Type.Fields }}
		}
		{{- end}}
	`, w, g.AdjCfg, struct {
		Type                       *schema.TypeStruct
		HaveOptionals              bool
		HaveTrailingOptionals      bool
		BeginTrailingOptionalField int
	}{
		g.Type,
		haveOptionals,
		haveTrailingOptionals,
		beginTrailingOptionalField,
	})
}

func (g structReprMapReprGenerator) EmitNodeMethodLength(w io.Writer) {
	// This is fun: it has to count down for any unset optional fields.
	// TODO : support for implicits is still future work.
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

func (g structReprMapReprGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g structReprMapReprGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g structReprMapReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return structReprMapReprBuilderGenerator{
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

type structReprMapReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeStruct
}

func (structReprMapReprBuilderGenerator) IsRepr() bool { return true } // hint used in some generalized templates.

func (g structReprMapReprBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the **m**aybe which communicates our completeness to the parent if we're a child assembler.
	// - 'state' is what it says on the tin.  this is used for the map state (the broad transitions between null, start-map, and finish are handled by 'm' for consistency.)
	// - 's' is a bitfield for what's been **s**et.
	// - 'f' is the **f**ocused field that will be assembled next.
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children that aren't allowed to be nullable (for those that are, their own maybe.m is used).
	// - the 'ca_*' fields embed **c**hild **a**ssemblers -- these are embedded so we can yield pointers to them without causing new allocations.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState
			s int
			f int

			cm schema.Maybe
			{{range $field := .Type.Fields -}}
			ca_{{ $field | FieldSymbolLower }} _{{ $field.Type | TypeSymbol }}__ReprAssembler
			{{end -}}
		}

		func (na *_{{ .Type | TypeSymbol }}__ReprAssembler) reset() {
			na.state = maState_initial
			na.s = 0
			{{- range $field := .Type.Fields }}
			na.ca_{{ $field | FieldSymbolLower }}.reset()
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_strictoid(w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	//
	// We do not set m=midvalue in phase 3 -- it shouldn't matter unless you're trying to pull off concurrent access, which is wrong and unsafe regardless.
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
func (g structReprMapReprBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	g.emitMapAssemblerChildTidyHelper(w)
	g.emitMapAssemblerMethods(w)
	g.emitKeyAssembler(w)
}
func (g structReprMapReprBuilderGenerator) emitMapAssemblerChildTidyHelper(w io.Writer) {
	// This is exactly the same as the matching method on the type-level assembler;
	//  everything that differs happens to be hidden behind the 'f' indirection, which is numeric.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) valueFinishTidy() bool {
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
					{{- /* while defense in depth here might avoid some 'wat' outcomes, it's not strictly necessary for safety */ -}}
					{{- /* ma.ca_{{ $field | FieldSymbolLower }}.w = nil */ -}}
					{{- /* ma.ca_{{ $field | FieldSymbolLower }}.m = nil */ -}}
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
func (g structReprMapReprBuilderGenerator) emitMapAssemblerMethods(w io.Writer) {
	// FUTURE: some of the setup of the child assemblers could probably be DRY'd up.
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
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: AssembleEntry cannot be called on an assembler that's already finished")
			}
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- if .Type.Fields }}
			switch k {
			{{- range $i, $field := .Type.Fields }}
			case "{{ $field | $type.RepresentationStrategy.GetFieldKey }}":
				if ma.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} != 0 {
					return nil, datamodel.ErrRepeatedMapKey{Key: &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}_serial}
				}
				ma.s += fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}
				ma.state = maState_midValue
				ma.f = {{ $i }}
				{{- if $field.IsMaybe }}
				ma.ca_{{ $field | FieldSymbolLower }}.w = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}ma.w.{{ $field | FieldSymbolLower }}.v
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.w.{{ $field | FieldSymbolLower }}.m
				{{if $field.IsNullable }}ma.w.{{ $field | FieldSymbolLower }}.m = allowNull{{end}}
				{{- else}}
				ma.ca_{{ $field | FieldSymbolLower }}.w = &ma.w.{{ $field | FieldSymbolLower }}
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.cm
				{{- end}}
				return &ma.ca_{{ $field | FieldSymbolLower }}, nil
			{{- end}}
			default:
			}
			{{- end}}
			return nil, schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Key:&_String{k}}
		}
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
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: AssembleKey cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midKey
			return (*_{{ .Type | TypeSymbol }}__ReprKeyAssembler)(ma)
		}
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
			switch ma.f {
			{{- range $i, $field := .Type.Fields }}
			case {{ $i }}:
				{{- if $field.IsMaybe }}
				ma.ca_{{ $field | FieldSymbolLower }}.w = {{if not (MaybeUsesPtr $field.Type) }}&{{end}}ma.w.{{ $field | FieldSymbolLower }}.v
				ma.ca_{{ $field | FieldSymbolLower }}.m = &ma.w.{{ $field | FieldSymbolLower }}.m
				{{if $field.IsNullable }}ma.w.{{ $field | FieldSymbolLower }}.m = allowNull{{end}}
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
			if ma.s & fieldBits__{{ $type | TypeSymbol }}_sufficient != fieldBits__{{ $type | TypeSymbol }}_sufficient {
				err := schema.ErrMissingRequiredField{Missing: make([]string, 0)}
				{{- range $i, $field := .Type.Fields }}
				{{- if not $field.IsMaybe}}
				if ma.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} == 0 {
					{{- if $field | $type.RepresentationStrategy.FieldHasRename }}
					err.Missing = append(err.Missing, "{{ $field.Name }} (serial:\"{{ $field | $type.RepresentationStrategy.GetFieldKey }}\")")
					{{- else}}
					err.Missing = append(err.Missing, "{{ $field.Name }}")
					{{- end}}
				}
				{{- end}}
				{{- end}}
				return err
			}
			ma.state = maState_finished
			*ma.m = schema.Maybe_Value
			return nil
		}
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) KeyPrototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
		func (ma *_{{ .Type | TypeSymbol }}__ReprAssembler) ValuePrototype(k string) datamodel.NodePrototype {
			panic("todo structbuilder mapassembler repr valueprototype")
		}
	`, w, g.AdjCfg, g)
}
func (g structReprMapReprBuilderGenerator) emitKeyAssembler(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprKeyAssembler _{{ .Type | TypeSymbol }}__ReprAssembler
	`, w, g.AdjCfg, g)
	stubs := mixins.StringAssemblerTraits{
		PkgName:       g.PkgName,
		TypeName:      g.TypeName + ".KeyAssembler", // ".Repr" is already in `g.TypeName`, so don't stutter the "Repr" part.
		AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__ReprKey",
	}
	// This key assembler can disregard any idea of complex keys because it's at the representation level!
	//  Map keys must always be plain strings at the representation level.
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
			{{- if .Type.Fields }}
			switch k {
			{{- $type := .Type -}} {{- /* ranging modifies dot, unhelpfully */ -}}
			{{- range $i, $field := .Type.Fields }}
			case "{{ $field | $type.RepresentationStrategy.GetFieldKey }}":
				if ka.s & fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }} != 0 {
					return datamodel.ErrRepeatedMapKey{Key: &fieldName__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}_serial}
				}
				ka.s += fieldBit__{{ $type | TypeSymbol }}_{{ $field | FieldSymbolUpper }}
				ka.state = maState_expectValue
				ka.f = {{ $i }}
				return nil
			{{- end }}
			}
			{{- end }}
			return schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}.Repr", Key:&_String{k}}
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
