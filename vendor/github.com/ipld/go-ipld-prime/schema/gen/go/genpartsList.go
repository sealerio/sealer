package gengo

import (
	"io"
)

// FIXME docs: these methods all say "-oid" but I think that was overoptimistic and not actually that applicable, really.
//  AssignNode?  Okay, that one's fine.
//  The rest?  They're all *very* emphatic about knowing either:
//   - that na.w.x is a slice; or,
//   - that there's only one 'va' (one type; and that it's reused).
//   The reuse level for those two traits is pretty minimal.

func emitNodeAssemblerMethodBeginList_listoid(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	//
	// There's surprisingly little variation for IsRepr on this one:
	//   - the child types we *store* are the same either way, so that doesn't vary;
	//   - the only thing that we return that's different is... ourself.
	//
	// DRY: even further, to an extracted function in the final output?  Maybe.
	//  This could be plausible, iff... the top half of the struct (na.m, na.w) was independently addressable.  (na.va has a varying concrete type and blocks extractions.)
	//  Would also want to examine if that makes desirable trades in gsloc/asmsize/speed/debuggability.
	//  Only seems to apply to case of list-repr-list, so unclear if worth the effort.
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
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
			if sizeHint > 0 {
				na.w.x = make([]_{{ .Type.ValueType | TypeSymbol }}{{if .Type.ValueIsNullable }}__Maybe{{end}}, 0, sizeHint)
			}
			return na, nil
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerMethodAssignNode_listoid(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	//
	// We do not set m=midvalue in phase 3 -- it shouldn't matter unless you're trying to pull off concurrent access, which is wrong and unsafe regardless.
	//
	// This works easily for both type-level and representational nodes because
	//  any divergences that have to do with the child value are nicely hidden behind `AssembleValue`.
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
			if v.Kind() != datamodel.Kind_List {
				return datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}{{ if .IsRepr }}.Repr{{end}}", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustList, ActualKind: v.Kind()}
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
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerHelper_listoid_tidyHelper(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// This function attempts to clean up the state machine to acknolwedge child value assembly finish.
	//  If the child was finished and we just collected it, return true and update state to laState_initial.
	//  Otherwise, if it wasn't done, return false;
	//   and the caller is almost certain to emit an error momentarily.
	// The function will only be called when the current state is laState_midValue.
	//  (In general, the idea is that if the user is doing things correctly,
	//   this function will only be called when the child is in fact finished.)
	// If 'cm' is used, we reset it to its initial condition of Maybe_Absent here.
	//  At the same time, we nil the 'w' pointer for the child assembler; otherwise its own state machine would probably let it modify 'w' again!
	//
	// DRY(nope): Can this be extracted to be a shared function between repr and type level nodes?
	//  It is textually identical, so... yeah, that'd be nice.  But...
	//  Nope.  It touches `la.va` directly.
	//   Attempting to extract that or hide it behind an interface would create virtual function calls in a very tight spot, and we don't want the execution time cost.
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) valueFinishTidy() bool {
			{{- if .Type.ValueIsNullable }}
			row := &la.w.x[len(la.w.x)-1]
			switch row.m {
			case schema.Maybe_Value:
				{{- if (MaybeUsesPtr .Type.ValueType) }}
				row.v = la.va.w
				{{- end}}
				la.va.w = nil
				fallthrough
			case schema.Maybe_Null:
				la.state = laState_initial
				la.va.reset()
				return true
			{{- else}}
			switch la.cm {
			case schema.Maybe_Value:
				la.va.w = nil
				la.cm = schema.Maybe_Absent
				la.state = laState_initial
				la.va.reset()
				return true
			{{- end}}
			default:
				return false
			}
		}
	`, w, adjCfg, data)
}

func emitNodeAssemblerHelper_listoid_listAssemblerMethods(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// DRY: Might want to split this up a bit further so it can be used by more kinds.
	//  Some parts of this could be reused by struct-repr-tuple, potentially, but would require being able to insert some more checks relating to length.
	//   This would also require excluding *all* 'va' references; those are radicaly different for structs, in that there's not even one (singular) of them.
	//
	// DRY(nope): Can this be extracted to a shared function in the output?
	//  Same story as the tidy helper -- it touches `la.va` concretely in several places, and that blocks extraction.
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) AssembleValue() datamodel.NodeAssembler {
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
			la.w.x = append(la.w.x, _{{ .Type.ValueType | TypeSymbol }}{{if .Type.ValueIsNullable }}__Maybe{{end}}{})
			la.state = laState_midValue
			row := &la.w.x[len(la.w.x)-1]
			{{- if .Type.ValueIsNullable }}
			{{- if not (MaybeUsesPtr .Type.ValueType) }}
			la.va.w = &row.v
			{{- end}}
			la.va.m = &row.m
			row.m = allowNull
			{{- else}}
			la.va.w = row
			la.va.m = &la.cm
			{{- end}}
			return &la.va
		}
	`, w, adjCfg, data)
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) Finish() error {
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
	`, w, adjCfg, data)
	doTemplate(`
		func (la *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) ValuePrototype(_ int64) datamodel.NodePrototype {
			return _{{ .Type.ValueType | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Prototype{}
		}
	`, w, adjCfg, data)
}
