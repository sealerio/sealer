package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &stringReprStringGenerator{}

func NewStringReprStringGenerator(pkgName string, typ *schema.TypeString, adjCfg *AdjunctCfg) TypeGenerator {
	return stringReprStringGenerator{
		stringGenerator{
			adjCfg,
			mixins.StringTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type stringReprStringGenerator struct {
	stringGenerator
}

func (g stringReprStringGenerator) GetRepresentationNodeGen() NodeGenerator {
	return stringReprStringReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type stringReprStringReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeString
}

func (g stringReprStringReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g stringReprStringReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (stringReprStringReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (stringReprStringReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (stringReprStringReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (stringReprStringReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (stringReprStringReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (stringReprStringReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (stringReprStringReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (stringReprStringReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (stringReprStringReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (stringReprStringReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (stringReprStringReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (stringReprStringReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (stringReprStringReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (stringReprStringReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (stringReprStringReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (stringReprStringReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (stringReprStringReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g stringReprStringReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g stringReprStringReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return stringReprStringReprBuilderGenerator(g)
}

type stringReprStringReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeString
}

func (stringReprStringReprBuilderGenerator) EmitNodeBuilderType(io.Writer)        {}
func (g stringReprStringReprBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {}
func (g stringReprStringReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (stringReprStringReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
