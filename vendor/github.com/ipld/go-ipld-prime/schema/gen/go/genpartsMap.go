package gengo

import (
	"io"
)

// FIXME docs: these methods all say "-oid" but I think that was overoptimistic and not actually that applicable, really.
//  AssignNode?  Okay, that one's fine.
//  The rest?  They're all *very* emphatic about knowing either:
//   - that na.w.t and na.w.m are fields; or,
//   - that there's only one 'ka' and 'va' (one type each; and that it's reused).
//   The reuse level for those two traits is pretty minimal.

func emitNodeAssemblerMethodBeginMap_mapoid(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
			switch *na.m {
			case schema.Maybe_Value, schema.Maybe_Null:
				panic("invalid state: cannot assign into assembler that's already finished")
			case midvalue:
				panic("invalid state: it makes no sense to 'begin' twice on the same assembler!")
			}
			*na.m = midvalue
			if sizeHint < 0 {
				sizeHint = 0
			}
			{{- if .Type | MaybeUsesPtr }}
			if na.w == nil {
				na.w = &_{{ .Type | TypeSymbol }}{}
			}
			{{- end}}
			na.w.m = make(map[_{{ .Type.KeyType | TypeSymbol }}]{{if .Type.ValueIsNullable }}Maybe{{else}}*_{{end}}{{ .Type.ValueType | TypeSymbol }}, sizeHint)
			na.w.t = make([]_{{ .Type | TypeSymbol }}__entry, 0, sizeHint)
			return na, nil
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerMethodAssignNode_mapoid(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	//
	// We do not set m=midvalue in phase 3 -- it shouldn't matter unless you're trying to pull off concurrent access, which is wrong and unsafe regardless.
	//
	// This works easily for both type-level and representational nodes because
	//  any divergences that have to do with the child value are nicely hidden behind  `AssembleKey` and `AssembleValue`.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssignNode(v datamodel.Node) error {
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
				return datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}{{ if .IsRepr }}.Repr{{end}}", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustMap, ActualKind: v.Kind()}
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
	`, w, adjCfg, data)
}

func emitNodeAssemblerHelper_mapoid_keyTidyHelper(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This function attempts to clean up the state machine to acknolwedge key assembly finish.
	//  If the child was finished and we just collected it, return true and update state to maState_expectValue.
	//   Collecting the child includes updating the 'ma.w.m' to point into the relevant row of 'ma.w.t', since that couldn't be done earlier,
	//    AND initializing the 'ma.va' (since we're already holding relevant offsets into 'ma.w.t').
	//  Otherwise, if it wasn't done, return false;
	//   and the caller is almost certain to emit an error momentarily.
	// The function will only be called when the current state is maState_midKey.
	//  (In general, the idea is that if the user is doing things correctly,
	//   this function will only be called when the child is in fact finished.)
	// Completion info always comes via 'cm', and we reset it to its initial condition of Maybe_Absent here.
	//  At the same time, we nil the 'w' pointer for the child assembler; otherwise its own state machine would probably let it modify 'w' again!
	//
	// DRY(nope): Can this be extracted to be a shared function between repr and type level nodes?
	//  It is textually identical, so... yeah, that'd be nice.  But...
	//  Nope.  It touches `ma.ka` and `ma.va` directly.
	//   Attempting to extract or hide those behind an interface would create virtual function calls in a very tight spot, and we don't want the execution time cost.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) keyFinishTidy() bool {
			switch ma.cm {
			case schema.Maybe_Value:
				ma.ka.w = nil
				tz := &ma.w.t[len(ma.w.t)-1]
				ma.cm = schema.Maybe_Absent
				ma.state = maState_expectValue
				ma.w.m[tz.k] = &tz.v
				{{- if .Type.ValueIsNullable }}
				{{- if not (MaybeUsesPtr .Type.ValueType) }}
				ma.va.w = &tz.v.v
				{{- end}}
				ma.va.m = &tz.v.m
				tz.v.m = allowNull
				{{- else}}
				ma.va.w = &tz.v
				ma.va.m = &ma.cm
				{{- end}}
				ma.ka.reset()
				return true
			default:
				return false
			}
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerHelper_mapoid_valueTidyHelper(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This function attempts to clean up the state machine to acknolwedge child value assembly finish.
	//  If the child was finished and we just collected it, return true and update state to maState_initial.
	//  Otherwise, if it wasn't done, return false;
	//   and the caller is almost certain to emit an error momentarily.
	// The function will only be called when the current state is maState_midValue.
	//  (In general, the idea is that if the user is doing things correctly,
	//   this function will only be called when the child is in fact finished.)
	// If 'cm' is used, we reset it to its initial condition of Maybe_Absent here.
	//  At the same time, we nil the 'w' pointer for the child assembler; otherwise its own state machine would probably let it modify 'w' again!
	//
	// DRY(nope): Can this be extracted to be a shared function between repr and type level nodes?
	//  Exact same story as the key tidy helper -- touches child assemblers concretely, and that blocks extraction.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) valueFinishTidy() bool {
			{{- if .Type.ValueIsNullable }}
			tz := &ma.w.t[len(ma.w.t)-1]
			switch tz.v.m {
			case schema.Maybe_Null:
				ma.state = maState_initial
				ma.va.reset()
				return true
			case schema.Maybe_Value:
				{{- if (MaybeUsesPtr .Type.ValueType) }}
				tz.v.v = ma.va.w
				{{- end}}
				ma.va.w = nil
				ma.state = maState_initial
				ma.va.reset()
				return true
			{{- else}}
			switch ma.cm {
			case schema.Maybe_Value:
				ma.va.w = nil
				ma.cm = schema.Maybe_Absent
				ma.state = maState_initial
				ma.va.reset()
				return true
			{{- end}}
			default:
				return false
			}
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerHelper_mapoid_mapAssemblerMethods(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// FUTURE: some of the setup of the child assemblers could probably be DRY'd up.
	//
	// REVIEW: there's a copy-by-value of k2 that's avoidable.  But it simplifies the error path.  Worth working on?
	//
	// REVIEW: processing the key via the reprPrototype of the key even when we're at the type level if it's type kind isn't string is currently supported, but should it be?  or is that more confusing than valuable?
	//  Very possible that it shouldn't be supported: the full-on keyAssembler route won't accept this, so consistency with that might be best.
	//  On the other hand, lookups by string *do* support this kind of processing (and it must, or PathSegment utility becomes unacceptably damaged), so either way, something feels surprising.
	//
	// DRY(nope): Can this be extracted to a shared function in the output?
	//  Same story as the tidy helpers -- it touches `va` and `ka` concretely in several places, and that blocks extraction.
	//
	// DRY: a lot of the state transition fences again are common for all mapoids, and could probably even be a function over '*state'...
	//   except for the fact they need to call the valueFinishTidy function, which is another one of those points that blocks extraction because we strongly don't want virtual functions calls there.
	//   Maybe the templates can be textually dedup'd more, though, at least.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
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

			var k2 _{{ .Type.KeyType | TypeSymbol }}
			{{- if or (not (eq .Type.KeyType.TypeKind.String "String")) .IsRepr }}
			if err := (_{{ .Type.KeyType | TypeSymbol }}__ReprPrototype{}).fromString(&k2, k); err != nil {
				return nil, err // TODO wrap in some kind of ErrInvalidKey
			}
			{{- else}}
			if err := (_{{ .Type.KeyType | TypeSymbol }}__Prototype{}).fromString(&k2, k); err != nil {
				return nil, err // TODO wrap in some kind of ErrInvalidKey
			}
			{{- end}}
			if _, exists := ma.w.m[k2]; exists {
				return nil, datamodel.ErrRepeatedMapKey{Key: &k2}
			}
			ma.w.t = append(ma.w.t, _{{ .Type | TypeSymbol }}__entry{k: k2})
			tz := &ma.w.t[len(ma.w.t)-1]
			ma.state = maState_midValue

			ma.w.m[k2] = &tz.v
			{{- if .Type.ValueIsNullable }}
			{{- if not (MaybeUsesPtr .Type.ValueType) }}
			ma.va.w = &tz.v.v
			{{- end}}
			ma.va.m = &tz.v.m
			tz.v.m = allowNull
			{{- else}}
			ma.va.w = &tz.v
			ma.va.m = &ma.cm
			{{- end}}
			return &ma.va, nil
		}
	`, w, adjCfg, data)
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssembleKey() datamodel.NodeAssembler {
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
			ma.w.t = append(ma.w.t, _{{ .Type | TypeSymbol }}__entry{})
			ma.state = maState_midKey
			ma.ka.m = &ma.cm
			ma.ka.w = &ma.w.t[len(ma.w.t)-1].k
			return &ma.ka
		}
	`, w, adjCfg, data)
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssembleValue() datamodel.NodeAssembler {
			switch ma.state {
			case maState_initial:
				panic("invalid state: AssembleValue cannot be called when no key is primed")
			case maState_midKey:
				if !ma.keyFinishTidy() {
					panic("invalid state: AssembleValue cannot be called when in the middle of assembling a key")
				} // if tidy success: carry on
			case maState_expectValue:
				// carry on
			case maState_midValue:
				panic("invalid state: AssembleValue cannot be called when in the middle of assembling another value")
			case maState_finished:
				panic("invalid state: AssembleValue cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midValue
			return &ma.va
		}
	`, w, adjCfg, data)
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) Finish() error {
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
			ma.state = maState_finished
			*ma.m = schema.Maybe_Value
			return nil
		}
	`, w, adjCfg, data)
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) KeyPrototype() datamodel.NodePrototype {
			return _{{ .Type.KeyType | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Prototype{}
		}
		func (ma *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) ValuePrototype(_ string) datamodel.NodePrototype {
			return _{{ .Type.ValueType | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Prototype{}
		}
	`, w, adjCfg, data)
}
