package gengo

import (
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime/testutil"
)

// EmitInternalEnums creates a file with enum types used internally.
// For example, the state machine values used in map and list builders.
// These always need to exist exactly once in each package created by codegen.
//
// The file header and import statements are included in the output of this function.
// (The imports in this file are different than most others in codegen output;
// we gather up any references to other packages in this file in order to simplify the rest of codegen's awareness of imports.)
func EmitInternalEnums(packageName string, w io.Writer) {
	fmt.Fprint(w, testutil.Dedent(`
		package `+packageName+`

		`+doNotEditComment+`

		import (
			"fmt"

			"github.com/ipld/go-ipld-prime/datamodel"
			"github.com/ipld/go-ipld-prime/schema"
		)

	`))

	// The 'Maybe' enum does double-duty in this package as a state machine for assembler completion.
	//
	// The 'Maybe_Absent' value gains the additional semantic of "clear to assign (but not null)"
	//  (which works because if you're *in* a value assembler, "absent" as a final result is already off the table).
	// Additionally, we get a few extra states that we cram into the same area of bits:
	//   - `midvalue` is used by assemblers of recursives to block AssignNull after BeginX.
	//   - `allowNull` is used by parent assemblers when initializing a child assembler to tell the child a transition to Maybe_Null is allowed in this context.
	fmt.Fprint(w, testutil.Dedent(`
		const (
			midvalue = schema.Maybe(4)
			allowNull = schema.Maybe(5)
		)

	`))

	fmt.Fprint(w, testutil.Dedent(`
		type maState uint8

		const (
			maState_initial     maState = iota
			maState_midKey
			maState_expectValue
			maState_midValue
			maState_finished
		)

		type laState uint8

		const (
			laState_initial  laState = iota
			laState_midValue
			laState_finished
		)
	`))

	// We occasionally need this erroring thunk to be able to snake an error out from some assembly processes.
	// It implements all of datamodel.NodeAssembler, but all of its methods return errors when used.
	fmt.Fprint(w, testutil.Dedent(`
		type _ErrorThunkAssembler struct {
			e error
		}

		func (ea _ErrorThunkAssembler) BeginMap(_ int64) (datamodel.MapAssembler, error) { return nil, ea.e }
		func (ea _ErrorThunkAssembler) BeginList(_ int64) (datamodel.ListAssembler, error) { return nil, ea.e }
		func (ea _ErrorThunkAssembler) AssignNull() error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignBool(bool) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignInt(int64) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignFloat(float64) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignString(string) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignBytes([]byte) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignLink(datamodel.Link) error { return ea.e }
		func (ea _ErrorThunkAssembler) AssignNode(datamodel.Node) error { return ea.e }
		func (ea _ErrorThunkAssembler) Prototype() datamodel.NodePrototype {
			panic(fmt.Errorf("cannot get prototype from error-carrying assembler: already derailed with error: %w", ea.e))
		}
	`))
}
