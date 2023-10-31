package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &boolReprBoolGenerator{}

func NewBoolReprBoolGenerator(pkgName string, typ *schema.TypeBool, adjCfg *AdjunctCfg) TypeGenerator {
	return boolReprBoolGenerator{
		boolGenerator{
			adjCfg,
			mixins.BoolTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type boolReprBoolGenerator struct {
	boolGenerator
}

func (g boolReprBoolGenerator) GetRepresentationNodeGen() NodeGenerator {
	return boolReprBoolReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type boolReprBoolReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeBool
}

func (g boolReprBoolReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g boolReprBoolReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (boolReprBoolReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (boolReprBoolReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (boolReprBoolReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (boolReprBoolReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (boolReprBoolReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (boolReprBoolReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (boolReprBoolReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (boolReprBoolReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (boolReprBoolReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (boolReprBoolReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (boolReprBoolReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (boolReprBoolReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g boolReprBoolReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g boolReprBoolReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return boolReprBoolReprBuilderGenerator(g)
}

type boolReprBoolReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeBool
}

func (boolReprBoolReprBuilderGenerator) EmitNodeBuilderType(io.Writer)    {}
func (boolReprBoolReprBuilderGenerator) EmitNodeBuilderMethods(io.Writer) {}
func (g boolReprBoolReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (boolReprBoolReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
