package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &linkReprLinkGenerator{}

func NewLinkReprLinkGenerator(pkgName string, typ *schema.TypeLink, adjCfg *AdjunctCfg) TypeGenerator {
	return linkReprLinkGenerator{
		linkGenerator{
			adjCfg,
			mixins.LinkTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type linkReprLinkGenerator struct {
	linkGenerator
}

func (g linkReprLinkGenerator) GetRepresentationNodeGen() NodeGenerator {
	return linkReprLinkReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type linkReprLinkReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeLink
}

func (g linkReprLinkReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g linkReprLinkReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (linkReprLinkReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (linkReprLinkReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (linkReprLinkReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (linkReprLinkReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (linkReprLinkReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (linkReprLinkReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (linkReprLinkReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (linkReprLinkReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (linkReprLinkReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (linkReprLinkReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (linkReprLinkReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (linkReprLinkReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g linkReprLinkReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g linkReprLinkReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return linkReprLinkReprBuilderGenerator(g)
}

type linkReprLinkReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeLink
}

func (linkReprLinkReprBuilderGenerator) EmitNodeBuilderType(io.Writer)    {}
func (linkReprLinkReprBuilderGenerator) EmitNodeBuilderMethods(io.Writer) {}
func (g linkReprLinkReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (linkReprLinkReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
