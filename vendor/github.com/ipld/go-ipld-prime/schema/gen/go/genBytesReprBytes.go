package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

var _ TypeGenerator = &bytesReprBytesGenerator{}

func NewBytesReprBytesGenerator(pkgName string, typ *schema.TypeBytes, adjCfg *AdjunctCfg) TypeGenerator {
	return bytesReprBytesGenerator{
		bytesGenerator{
			adjCfg,
			mixins.BytesTraits{
				PkgName:    pkgName,
				TypeName:   string(typ.Name()),
				TypeSymbol: adjCfg.TypeSymbol(typ),
			},
			pkgName,
			typ,
		},
	}
}

type bytesReprBytesGenerator struct {
	bytesGenerator
}

func (g bytesReprBytesGenerator) GetRepresentationNodeGen() NodeGenerator {
	return bytesReprBytesReprGenerator{
		g.AdjCfg,
		g.Type,
	}
}

type bytesReprBytesReprGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeBytes
}

func (g bytesReprBytesReprGenerator) EmitNodeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Repr = _{{ .Type | TypeSymbol }}
	`, w, g.AdjCfg, g)
}
func (g bytesReprBytesReprGenerator) EmitNodeTypeAssertions(w io.Writer) {
	doTemplate(`
		var _ datamodel.Node = &_{{ .Type | TypeSymbol }}__Repr{}
	`, w, g.AdjCfg, g)
}
func (bytesReprBytesReprGenerator) EmitNodeMethodKind(io.Writer)            {}
func (bytesReprBytesReprGenerator) EmitNodeMethodLookupByString(io.Writer)  {}
func (bytesReprBytesReprGenerator) EmitNodeMethodLookupByNode(io.Writer)    {}
func (bytesReprBytesReprGenerator) EmitNodeMethodLookupByIndex(io.Writer)   {}
func (bytesReprBytesReprGenerator) EmitNodeMethodLookupBySegment(io.Writer) {}
func (bytesReprBytesReprGenerator) EmitNodeMethodMapIterator(io.Writer)     {}
func (bytesReprBytesReprGenerator) EmitNodeMethodListIterator(io.Writer)    {}
func (bytesReprBytesReprGenerator) EmitNodeMethodLength(io.Writer)          {}
func (bytesReprBytesReprGenerator) EmitNodeMethodIsAbsent(io.Writer)        {}
func (bytesReprBytesReprGenerator) EmitNodeMethodIsNull(io.Writer)          {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsBool(io.Writer)          {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsInt(io.Writer)           {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsFloat(io.Writer)         {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsString(io.Writer)        {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsBytes(io.Writer)         {}
func (bytesReprBytesReprGenerator) EmitNodeMethodAsLink(io.Writer)          {}
func (bytesReprBytesReprGenerator) EmitNodeMethodPrototype(io.Writer)       {}
func (g bytesReprBytesReprGenerator) EmitNodePrototypeType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprPrototype = _{{ .Type | TypeSymbol }}__Prototype
	`, w, g.AdjCfg, g)
}
func (g bytesReprBytesReprGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return bytesReprBytesReprBuilderGenerator(g)
}

type bytesReprBytesReprBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	Type   *schema.TypeBytes
}

func (bytesReprBytesReprBuilderGenerator) EmitNodeBuilderType(io.Writer)    {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeBuilderMethods(io.Writer) {}
func (g bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Since this is a "natural" representation... there's just a type alias here.
	//  No new functions are necessary.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__ReprAssembler = _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodBeginMap(io.Writer)     {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodBeginList(io.Writer)    {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignNull(io.Writer)   {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignBool(io.Writer)   {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignInt(io.Writer)    {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignFloat(io.Writer)  {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignString(io.Writer) {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignBytes(io.Writer)  {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignLink(io.Writer)   {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodAssignNode(io.Writer)   {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerMethodPrototype(io.Writer)    {}
func (bytesReprBytesReprBuilderGenerator) EmitNodeAssemblerOtherBits(io.Writer)          {}
