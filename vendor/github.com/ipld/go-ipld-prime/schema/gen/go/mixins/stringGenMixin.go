package mixins

import (
	"io"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type StringTraits struct {
	PkgName    string
	TypeName   string // see doc in kindTraitsGenerator
	TypeSymbol string // see doc in kindTraitsGenerator
}

func (StringTraits) Kind() datamodel.Kind {
	return datamodel.Kind_String
}
func (g StringTraits) EmitNodeMethodKind(w io.Writer) {
	doTemplate(`
		func ({{ .TypeSymbol }}) Kind() datamodel.Kind {
			return datamodel.Kind_String
		}
	`, w, g)
}
func (g StringTraits) EmitNodeMethodLookupByString(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodLookupByString(w)
}
func (g StringTraits) EmitNodeMethodLookupByNode(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodLookupByNode(w)
}
func (g StringTraits) EmitNodeMethodLookupByIndex(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodLookupByIndex(w)
}
func (g StringTraits) EmitNodeMethodLookupBySegment(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodLookupBySegment(w)
}
func (g StringTraits) EmitNodeMethodMapIterator(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodMapIterator(w)
}
func (g StringTraits) EmitNodeMethodListIterator(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodListIterator(w)
}
func (g StringTraits) EmitNodeMethodLength(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodLength(w)
}
func (g StringTraits) EmitNodeMethodIsAbsent(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodIsAbsent(w)
}
func (g StringTraits) EmitNodeMethodIsNull(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodIsNull(w)
}
func (g StringTraits) EmitNodeMethodAsBool(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodAsBool(w)
}
func (g StringTraits) EmitNodeMethodAsInt(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodAsInt(w)
}
func (g StringTraits) EmitNodeMethodAsFloat(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodAsFloat(w)
}
func (g StringTraits) EmitNodeMethodAsBytes(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodAsBytes(w)
}
func (g StringTraits) EmitNodeMethodAsLink(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_String}.emitNodeMethodAsLink(w)
}

type StringAssemblerTraits struct {
	PkgName       string
	TypeName      string // see doc in kindAssemblerTraitsGenerator
	AppliedPrefix string // see doc in kindAssemblerTraitsGenerator
}

func (StringAssemblerTraits) Kind() datamodel.Kind {
	return datamodel.Kind_String
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodBeginMap(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodBeginList(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignNull(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignBool(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignBool(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignInt(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignInt(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignFloat(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignFloat(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignBytes(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignBytes(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodAssignLink(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodAssignLink(w)
}
func (g StringAssemblerTraits) EmitNodeAssemblerMethodPrototype(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_String}.emitNodeAssemblerMethodPrototype(w)
}
