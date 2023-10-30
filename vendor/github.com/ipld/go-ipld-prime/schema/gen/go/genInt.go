package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type intGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.IntTraits
	PkgName string
	Type    *schema.TypeInt
}

func (intGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g intGenerator) EmitNativeType(w io.Writer) {
	emitNativeType_scalar(w, g.AdjCfg, g)
}
func (g intGenerator) EmitNativeAccessors(w io.Writer) {
	emitNativeAccessors_scalar(w, g.AdjCfg, g)
}
func (g intGenerator) EmitNativeBuilder(w io.Writer) {
	emitNativeBuilder_scalar(w, g.AdjCfg, g)
}

func (g intGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g intGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g intGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g intGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g intGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
}
func (g intGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}
func (g intGenerator) EmitNodeMethodAsInt(w io.Writer) {
	emitNodeMethodAsKind_scalar(w, g.AdjCfg, g)
}
func (g intGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}
func (g intGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g intGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return intBuilderGenerator{
		g.AdjCfg,
		mixins.IntAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type intBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.IntAssemblerTraits
	PkgName string
	Type    *schema.TypeInt
}

func (intBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g intBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	emitNodeAssemblerType_scalar(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeAssemblerMethodAssignInt(w io.Writer) {
	emitNodeAssemblerMethodAssignKind_scalar(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_scalar(w, g.AdjCfg, g)
}
func (g intBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// Nothing needed here for int kinds.
}
