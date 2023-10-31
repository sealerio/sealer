package gengo

import (
	"io"
)

// What's "strictoid" mean?  Anything that's recursive but not infinite -- so structs, unions.

// This form of BeginMap method is reusable for anything that has fixed size and thus ignores sizehint.
// That means: structs, unions, and their relevant representations.
func emitNodeAssemblerMethodBeginMap_strictoid(w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	// We currently disregard sizeHint.  It's not relevant to us.
	//  We could check it strictly and emit errors; presently, we don't.
	// This method contains a branch to support MaybeUsesPtr because new memory may need to be allocated.
	//  This allocation only happens if the 'w' ptr is nil, which means we're being used on a Maybe;
	//  otherwise, the 'w' ptr should already be set, and we fill that memory location without allocating, as usual.
	//  (Note that the Maybe we're talking about here is for us, not our children: so it applies on unions too.)
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__{{ if .IsRepr }}Repr{{end}}Assembler) BeginMap(int64) (datamodel.MapAssembler, error) {
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
	`, w, adjCfg, data)
}
