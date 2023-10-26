package schemadmt

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/bindnode"
)

// ConcatenateSchemas returns a new schema DMT object containing the
// type declarations from both.
//
// As is usual for DMT form data, there is no check about the validity
// of the result yet; you'll need to apply `Compile` on the produced value
// to produce a usable compiled typesystem or to become certain that
// all references in the DMT are satisfied, etc.
func ConcatenateSchemas(a, b *Schema) *Schema {
	// The joy of having an intermediate form that's just regular data model:
	// we can implement this by simply using data model "copy" operations,
	// and the result is correct.
	nb := Prototypes.Schema.NewBuilder()
	if err := datamodel.Copy(bindnode.Wrap(a, Prototypes.Schema.Type()), nb); err != nil {
		panic(err)
	}
	if err := datamodel.Copy(bindnode.Wrap(b, Prototypes.Schema.Type()), nb); err != nil {
		panic(err)
	}
	return bindnode.Unwrap(nb.Build()).(*Schema)
}
