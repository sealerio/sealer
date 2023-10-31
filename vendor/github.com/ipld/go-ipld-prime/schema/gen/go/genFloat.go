package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type float64Generator struct {
	AdjCfg *AdjunctCfg
	mixins.FloatTraits
	PkgName string
	Type    *schema.TypeFloat
}

func (float64Generator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g float64Generator) EmitNativeType(w io.Writer) {
	emitNativeType_scalar(w, g.AdjCfg, g)
}
func (g float64Generator) EmitNativeAccessors(w io.Writer) {
	emitNativeAccessors_scalar(w, g.AdjCfg, g)
}
func (g float64Generator) EmitNativeBuilder(w io.Writer) {
	emitNativeBuilder_scalar(w, g.AdjCfg, g)
}

func (g float64Generator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g float64Generator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g float64Generator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g float64Generator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g float64Generator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
}
func (g float64Generator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}
func (g float64Generator) EmitNodeMethodAsFloat(w io.Writer) {
	emitNodeMethodAsKind_scalar(w, g.AdjCfg, g)
}
func (g float64Generator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}
func (g float64Generator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g float64Generator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return float64BuilderGenerator{
		g.AdjCfg,
		mixins.FloatAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type float64BuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.FloatAssemblerTraits
	PkgName string
	Type    *schema.TypeFloat
}

func (float64BuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g float64BuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	emitNodeAssemblerType_scalar(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeAssemblerMethodAssignFloat(w io.Writer) {
	emitNodeAssemblerMethodAssignKind_scalar(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_scalar(w, g.AdjCfg, g)
}
func (g float64BuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// Nothing needed here for float64 kinds.
}
