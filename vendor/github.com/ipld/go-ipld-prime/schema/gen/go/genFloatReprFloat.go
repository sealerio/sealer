package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &float64ReprFloatGenerator{}

func NewFloatReprFloatGenerator(pkgName string, typ *schema.TypeFloat, adjCfg *AdjunctCfg) TypeGenerator {
	return float64ReprFloatGenerator{
		float64Generator{
			adjCfg,
			mixins.FloatTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type float64ReprFloatGenerator struct {
	float64Generator
}

func (g float64ReprFloatGenerator) GetRepresentationNodeGen() NodeGenerator {
	return float64ReprFloatReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type float64ReprFloatReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeFloat
}

func (g float64ReprFloatReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g float64ReprFloatReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (float64ReprFloatReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (float64ReprFloatReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (float64ReprFloatReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (float64ReprFloatReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (float64ReprFloatReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (float64ReprFloatReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (float64ReprFloatReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (float64ReprFloatReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (float64ReprFloatReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (float64ReprFloatReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (float64ReprFloatReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (float64ReprFloatReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g float64ReprFloatReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g float64ReprFloatReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return float64ReprFloatReprBuilderGenerator(g)
}

type float64ReprFloatReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeFloat
}

func (float64ReprFloatReprBuilderGenerator) EmitNodeBuilderType(io.Writer)    {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeBuilderMethods(io.Writer) {}
func (g float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (float64ReprFloatReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
