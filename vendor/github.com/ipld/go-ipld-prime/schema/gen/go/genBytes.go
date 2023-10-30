package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type bytesGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.BytesTraits
	PkgName string
	Type    *schema.TypeBytes
}

func (bytesGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g bytesGenerator) EmitNativeType(w io.Writer) {
	emitNativeType_scalar(w, g.AdjCfg, g)
}
func (g bytesGenerator) EmitNativeAccessors(w io.Writer) {
	emitNativeAccessors_scalar(w, g.AdjCfg, g)
}
func (g bytesGenerator) EmitNativeBuilder(w io.Writer) {
	emitNativeBuilder_scalar(w, g.AdjCfg, g)
}

func (g bytesGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g bytesGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g bytesGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g bytesGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g bytesGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
}
func (g bytesGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}
func (g bytesGenerator) EmitNodeMethodAsBytes(w io.Writer) {
	emitNodeMethodAsKind_scalar(w, g.AdjCfg, g)
}
func (g bytesGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}
func (g bytesGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g bytesGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return bytesBuilderGenerator{
		g.AdjCfg,
		mixins.BytesAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type bytesBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.BytesAssemblerTraits
	PkgName string
	Type    *schema.TypeBytes
}

func (bytesBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g bytesBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	emitNodeAssemblerType_scalar(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(w io.Writer) {
	emitNodeAssemblerMethodAssignKind_scalar(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_scalar(w, g.AdjCfg, g)
}
func (g bytesBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// Nothing needed here for bytes kinds.
}
