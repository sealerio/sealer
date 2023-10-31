package mixins

import (
	"io"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type BoolTraits struct {
	PkgName    string
	TypeName   string // see doc in kindTraitsGenerator
	TypeSymbol string // see doc in kindTraitsGenerator
}

func (BoolTraits) Kind() datamodel.Kind {
	return datamodel.Kind_Bool
}
func (g BoolTraits) EmitNodeMethodKind(w io.Writer) {
	doTemplate(`
		func ({{ .TypeSymbol }}) Kind() datamodel.Kind {
			return datamodel.Kind_Bool
		}
	`, w, g)
}
func (g BoolTraits) EmitNodeMethodLookupByString(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodLookupByString(w)
}
func (g BoolTraits) EmitNodeMethodLookupByNode(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodLookupByNode(w)
}
func (g BoolTraits) EmitNodeMethodLookupByIndex(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodLookupByIndex(w)
}
func (g BoolTraits) EmitNodeMethodLookupBySegment(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodLookupBySegment(w)
}
func (g BoolTraits) EmitNodeMethodMapIterator(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodMapIterator(w)
}
func (g BoolTraits) EmitNodeMethodListIterator(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodListIterator(w)
}
func (g BoolTraits) EmitNodeMethodLength(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodLength(w)
}
func (g BoolTraits) EmitNodeMethodIsAbsent(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodIsAbsent(w)
}
func (g BoolTraits) EmitNodeMethodIsNull(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodIsNull(w)
}
func (g BoolTraits) EmitNodeMethodAsInt(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodAsInt(w)
}
func (g BoolTraits) EmitNodeMethodAsFloat(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodAsFloat(w)
}
func (g BoolTraits) EmitNodeMethodAsString(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodAsString(w)
}
func (g BoolTraits) EmitNodeMethodAsBytes(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodAsBytes(w)
}
func (g BoolTraits) EmitNodeMethodAsLink(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Bool}.emitNodeMethodAsLink(w)
}

type BoolAssemblerTraits struct {
	PkgName       string
	TypeName      string // see doc in kindAssemblerTraitsGenerator
	AppliedPrefix string // see doc in kindAssemblerTraitsGenerator
}

func (BoolAssemblerTraits) Kind() datamodel.Kind {
	return datamodel.Kind_Bool
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodBeginMap(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodBeginList(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignNull(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignInt(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignInt(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignFloat(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignFloat(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignString(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignString(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignBytes(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignBytes(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodAssignLink(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodAssignLink(w)
}
func (g BoolAssemblerTraits) EmitNodeAssemblerMethodPrototype(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Bool}.emitNodeAssemblerMethodPrototype(w)
}
