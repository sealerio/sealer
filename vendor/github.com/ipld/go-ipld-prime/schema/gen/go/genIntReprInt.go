package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &intReprIntGenerator{}

func NewIntReprIntGenerator(pkgName string, typ *schema.TypeInt, adjCfg *AdjunctCfg) TypeGenerator {
	return intReprIntGenerator{
		intGenerator{
			adjCfg,
			mixins.IntTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type intReprIntGenerator struct {
	intGenerator
}

func (g intReprIntGenerator) GetRepresentationNodeGen() NodeGenerator {
	return intReprIntReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type intReprIntReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeInt
}

func (g intReprIntReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g intReprIntReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (intReprIntReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (intReprIntReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (intReprIntReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (intReprIntReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (intReprIntReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (intReprIntReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (intReprIntReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (intReprIntReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (intReprIntReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (intReprIntReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (intReprIntReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (intReprIntReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (intReprIntReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (intReprIntReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (intReprIntReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (intReprIntReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (intReprIntReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g intReprIntReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g intReprIntReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return intReprIntReprBuilderGenerator(g)
}

type intReprIntReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeInt
}

func (intReprIntReprBuilderGenerator) EmitNodeBuilderType(io.Writer)    {}
func (intReprIntReprBuilderGenerator) EmitNodeBuilderMethods(io.Writer) {}
func (g intReprIntReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (intReprIntReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
