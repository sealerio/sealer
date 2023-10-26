package fluent

import (
	"github.com/ipld/go-ipld-prime/datamodel"
)

func Build(np datamodel.NodePrototype, fn func(NodeAssembler)) (datamodel.Node, error) {
	nb := np.NewBuilder()
	fna := WrapAssembler(nb)
	err := Recover(func() {
		fn(fna)
	})
	if err != nil {
		return nil, err
	}
	return nb.Build(), nil
}
func BuildMap(np datamodel.NodePrototype, sizeHint int64, fn func(MapAssembler)) (datamodel.Node, error) {
	return Build(np, func(fna NodeAssembler) { fna.CreateMap(sizeHint, fn) })
}
func BuildList(np datamodel.NodePrototype, sizeHint int64, fn func(ListAssembler)) (datamodel.Node, error) {
	return Build(np, func(fna NodeAssembler) { fna.CreateList(sizeHint, fn) })
}

func MustBuild(np datamodel.NodePrototype, fn func(NodeAssembler)) datamodel.Node {
	nb := np.NewBuilder()
	fn(WrapAssembler(nb))
	return nb.Build()
}
func MustBuildMap(np datamodel.NodePrototype, sizeHint int64, fn func(MapAssembler)) datamodel.Node {
	return MustBuild(np, func(fna NodeAssembler) { fna.CreateMap(sizeHint, fn) })
}
func MustBuildList(np datamodel.NodePrototype, sizeHint int64, fn func(ListAssembler)) datamodel.Node {
	return MustBuild(np, func(fna NodeAssembler) { fna.CreateList(sizeHint, fn) })
}

func WrapAssembler(na datamodel.NodeAssembler) NodeAssembler {
	return &nodeAssembler{na}
}

// NodeAssembler is the same as the interface in the core package, except:
// instead of returning errors, any error will cause panic
// (and you can collect these with `fluent.Recover`);
// and all recursive operations take a function as a parameter,
// within which you will receive another {Map,List,}NodeAssembler.
type NodeAssembler interface {
	CreateMap(sizeHint int64, fn func(MapAssembler))
	CreateList(sizeHint int64, fn func(ListAssembler))
	AssignNull()
	AssignBool(bool)
	AssignInt(int64)
	AssignFloat(float64)
	AssignString(string)
	AssignBytes([]byte)
	AssignLink(datamodel.Link)
	AssignNode(datamodel.Node)

	Prototype() datamodel.NodePrototype
}

// MapAssembler is the same as the interface in the core package, except:
// instead of returning errors, any error will cause panic
// (and you can collect these with `fluent.Recover`);
// and all recursive operations take a function as a parameter,
// within which you will receive another {Map,List,}NodeAssembler.
type MapAssembler interface {
	AssembleKey() NodeAssembler
	AssembleValue() NodeAssembler

	AssembleEntry(k string) NodeAssembler

	KeyPrototype() datamodel.NodePrototype
	ValuePrototype(k string) datamodel.NodePrototype
}

// ListAssembler is the same as the interface in the core package, except:
// instead of returning errors, any error will cause panic
// (and you can collect these with `fluent.Recover`);
// and all recursive operations take a function as a parameter,
// within which you will receive another {Map,List,}NodeAssembler.
type ListAssembler interface {
	AssembleValue() NodeAssembler

	ValuePrototype(idx int64) datamodel.NodePrototype
}

type nodeAssembler struct {
	na datamodel.NodeAssembler
}

func (fna *nodeAssembler) CreateMap(sizeHint int64, fn func(MapAssembler)) {
	if ma, err := fna.na.BeginMap(sizeHint); err != nil {
		panic(Error{err})
	} else {
		fn(&mapNodeAssembler{ma})
		if err := ma.Finish(); err != nil {
			panic(Error{err})
		}
	}
}
func (fna *nodeAssembler) CreateList(sizeHint int64, fn func(ListAssembler)) {
	if la, err := fna.na.BeginList(sizeHint); err != nil {
		panic(Error{err})
	} else {
		fn(&listNodeAssembler{la})
		if err := la.Finish(); err != nil {
			panic(Error{err})
		}
	}
}
func (fna *nodeAssembler) AssignNull() {
	if err := fna.na.AssignNull(); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignBool(v bool) {
	if err := fna.na.AssignBool(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignInt(v int64) {
	if err := fna.na.AssignInt(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignFloat(v float64) {
	if err := fna.na.AssignFloat(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignString(v string) {
	if err := fna.na.AssignString(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignBytes(v []byte) {
	if err := fna.na.AssignBytes(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignLink(v datamodel.Link) {
	if err := fna.na.AssignLink(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) AssignNode(v datamodel.Node) {
	if err := fna.na.AssignNode(v); err != nil {
		panic(Error{err})
	}
}
func (fna *nodeAssembler) Prototype() datamodel.NodePrototype {
	return fna.na.Prototype()
}

type mapNodeAssembler struct {
	ma datamodel.MapAssembler
}

func (fma *mapNodeAssembler) AssembleKey() NodeAssembler {
	return &nodeAssembler{fma.ma.AssembleKey()}
}
func (fma *mapNodeAssembler) AssembleValue() NodeAssembler {
	return &nodeAssembler{fma.ma.AssembleValue()}
}
func (fma *mapNodeAssembler) AssembleEntry(k string) NodeAssembler {
	va, err := fma.ma.AssembleEntry(k)
	if err != nil {
		panic(Error{err})
	}
	return &nodeAssembler{va}
}
func (fma *mapNodeAssembler) KeyPrototype() datamodel.NodePrototype {
	return fma.ma.KeyPrototype()
}
func (fma *mapNodeAssembler) ValuePrototype(k string) datamodel.NodePrototype {
	return fma.ma.ValuePrototype(k)
}

type listNodeAssembler struct {
	la datamodel.ListAssembler
}

func (fla *listNodeAssembler) AssembleValue() NodeAssembler {
	return &nodeAssembler{fla.la.AssembleValue()}
}
func (fla *listNodeAssembler) ValuePrototype(idx int64) datamodel.NodePrototype {
	return fla.la.ValuePrototype(idx)
}
