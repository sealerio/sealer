package mixins

import (
	"io"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type MapTraits struct {
	PkgName    string
	TypeName   string // see doc in kindTraitsGenerator
	TypeSymbol string // see doc in kindTraitsGenerator
}

func (MapTraits) Kind() datamodel.Kind {
	return datamodel.Kind_Map
}
func (g MapTraits) EmitNodeMethodKind(w io.Writer) {
	doTemplate(`
		func ({{ .TypeSymbol }}) Kind() datamodel.Kind {
			return datamodel.Kind_Map
		}
	`, w, g)
}
func (g MapTraits) EmitNodeMethodLookupByIndex(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodLookupByIndex(w)
}
func (g MapTraits) EmitNodeMethodLookupBySegment(w io.Writer) {
	doTemplate(`
		func (n {{ .TypeSymbol }}) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
			return n.LookupByString(seg.String())
		}
	`, w, g)
}
func (g MapTraits) EmitNodeMethodListIterator(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodListIterator(w)
}
func (g MapTraits) EmitNodeMethodIsAbsent(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodIsAbsent(w)
}
func (g MapTraits) EmitNodeMethodIsNull(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodIsNull(w)
}
func (g MapTraits) EmitNodeMethodAsBool(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsBool(w)
}
func (g MapTraits) EmitNodeMethodAsInt(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsInt(w)
}
func (g MapTraits) EmitNodeMethodAsFloat(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsFloat(w)
}
func (g MapTraits) EmitNodeMethodAsString(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsString(w)
}
func (g MapTraits) EmitNodeMethodAsBytes(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsBytes(w)
}
func (g MapTraits) EmitNodeMethodAsLink(w io.Writer) {
	kindTraitsGenerator{g.PkgName, g.TypeName, g.TypeSymbol, datamodel.Kind_Map}.emitNodeMethodAsLink(w)
}

type MapAssemblerTraits struct {
	PkgName       string
	TypeName      string // see doc in kindAssemblerTraitsGenerator
	AppliedPrefix string // see doc in kindAssemblerTraitsGenerator
}

func (MapAssemblerTraits) Kind() datamodel.Kind {
	return datamodel.Kind_Map
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodBeginList(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodBeginList(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignNull(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignBool(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignBool(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignInt(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignInt(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignFloat(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignFloat(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignString(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignString(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignBytes(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignBytes(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodAssignLink(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodAssignLink(w)
}
func (g MapAssemblerTraits) EmitNodeAssemblerMethodPrototype(w io.Writer) {
	kindAssemblerTraitsGenerator{g.PkgName, g.TypeName, g.AppliedPrefix, datamodel.Kind_Map}.emitNodeAssemblerMethodPrototype(w)
}
