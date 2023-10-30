package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

type linkGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.LinkTraits
	PkgName string
	Type    *schema.TypeLink
}

func (linkGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g linkGenerator) EmitNativeType(w io.Writer) {
	emitNativeType_scalar(w, g.AdjCfg, g)
}
func (g linkGenerator) EmitNativeAccessors(w io.Writer) {
	emitNativeAccessors_scalar(w, g.AdjCfg, g)
}
func (g linkGenerator) EmitNativeBuilder(w io.Writer) {
	emitNativeBuilder_scalar(w, g.AdjCfg, g)
}

func (g linkGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g linkGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g linkGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)

	// Bonus feature for some links (conforms to the schema.TypedLinkNode interface):
	if g.Type.HasReferencedType() {
		doTemplate(`
			func ({{ .Type | TypeSymbol }}) LinkTargetNodePrototype() datamodel.NodePrototype {
				return Type.{{ .Type.ReferencedType | TypeSymbol }}__Repr
			}
		`, w, g.AdjCfg, g)
	}
}

func (g linkGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g linkGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.
}
func (g linkGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}
func (g linkGenerator) EmitNodeMethodAsLink(w io.Writer) {
	emitNodeMethodAsKind_scalar(w, g.AdjCfg, g)
}
func (g linkGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}
func (g linkGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g linkGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return linkBuilderGenerator{
		g.AdjCfg,
		mixins.LinkAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type linkBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.LinkAssemblerTraits
	PkgName string
	Type    *schema.TypeLink
}

func (linkBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g linkBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	emitNodeAssemblerType_scalar(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	emitNodeAssemblerMethodAssignNull_scalar(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeAssemblerMethodAssignLink(w io.Writer) {
	emitNodeAssemblerMethodAssignKind_scalar(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	emitNodeAssemblerMethodAssignNode_scalar(w, g.AdjCfg, g)
}
func (g linkBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	// Nothing needed here for link kinds.
}
