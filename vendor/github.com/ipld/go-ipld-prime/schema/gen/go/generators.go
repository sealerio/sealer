package gengo

import (
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime/schema"
)

// TypeGenerator gathers all the info for generating all code related to one
// type in the schema.
type TypeGenerator interface {
	// -- the natively-typed apis -->

	EmitNativeType(io.Writer)
	EmitNativeAccessors(io.Writer) // depends on the kind -- field accessors for struct, typed iterators for map, etc.
	EmitNativeBuilder(io.Writer)   // typically emits some kind of struct that has a Build method.
	EmitNativeMaybe(io.Writer)     // a pointer-free 'maybe' mechanism is generated for all types.

	// -- the schema.TypedNode.Type method and vars -->

	EmitTypeConst(io.Writer)           // these emit dummies for now
	EmitTypedNodeMethodType(io.Writer) // these emit dummies for now

	// -- all node methods -->
	//   (and note that the nodeBuilder for this one should be the "semantic" one,
	//     e.g. it *always* acts like a map for structs, even if the repr is different.)

	NodeGenerator

	// -- and the representation and its node and nodebuilder -->
	//    (these vary!)

	EmitTypedNodeMethodRepresentation(io.Writer)
	GetRepresentationNodeGen() NodeGenerator // includes transitively the matched NodeBuilderGenerator
}

type NodeGenerator interface {
	EmitNodeType(io.Writer)           // usually already covered by EmitNativeType for the primary node, but has a nonzero body for the repr node
	EmitNodeTypeAssertions(io.Writer) // optional to include this content
	EmitNodeMethodKind(io.Writer)
	EmitNodeMethodLookupByString(io.Writer)
	EmitNodeMethodLookupByNode(io.Writer)
	EmitNodeMethodLookupByIndex(io.Writer)
	EmitNodeMethodLookupBySegment(io.Writer)
	EmitNodeMethodMapIterator(io.Writer)  // also iterator itself
	EmitNodeMethodListIterator(io.Writer) // also iterator itself
	EmitNodeMethodLength(io.Writer)
	EmitNodeMethodIsAbsent(io.Writer)
	EmitNodeMethodIsNull(io.Writer)
	EmitNodeMethodAsBool(io.Writer)
	EmitNodeMethodAsInt(io.Writer)
	EmitNodeMethodAsFloat(io.Writer)
	EmitNodeMethodAsString(io.Writer)
	EmitNodeMethodAsBytes(io.Writer)
	EmitNodeMethodAsLink(io.Writer)
	EmitNodeMethodPrototype(io.Writer)
	EmitNodePrototypeType(io.Writer)
	GetNodeBuilderGenerator() NodeBuilderGenerator // assembler features also included inside
}

type NodeBuilderGenerator interface {
	EmitNodeBuilderType(io.Writer)
	EmitNodeBuilderMethods(io.Writer) // not many, so just slung them together.
	EmitNodeAssemblerType(io.Writer)  // you can call this and not EmitNodeBuilderType in some situations.
	EmitNodeAssemblerMethodBeginMap(io.Writer)
	EmitNodeAssemblerMethodBeginList(io.Writer)
	EmitNodeAssemblerMethodAssignNull(io.Writer)
	EmitNodeAssemblerMethodAssignBool(io.Writer)
	EmitNodeAssemblerMethodAssignInt(io.Writer)
	EmitNodeAssemblerMethodAssignFloat(io.Writer)
	EmitNodeAssemblerMethodAssignString(io.Writer)
	EmitNodeAssemblerMethodAssignBytes(io.Writer)
	EmitNodeAssemblerMethodAssignLink(io.Writer)
	EmitNodeAssemblerMethodAssignNode(io.Writer)
	EmitNodeAssemblerMethodPrototype(io.Writer)
	EmitNodeAssemblerOtherBits(io.Writer) // key and value child assemblers are done here.
}

// EmitFileHeader emits a baseline package header that will
// allow a file with a generated type to compile.
// (Fortunately, there are no variations in this.)
func EmitFileHeader(packageName string, w io.Writer) {
	fmt.Fprintf(w, "package %s\n\n", packageName)
	fmt.Fprintf(w, doNotEditComment+"\n\n")
	fmt.Fprintf(w, "import (\n")
	fmt.Fprintf(w, "\t\"github.com/ipld/go-ipld-prime/datamodel\"\n")
	fmt.Fprintf(w, "\t\"github.com/ipld/go-ipld-prime/node/mixins\"\n")
	fmt.Fprintf(w, "\t\"github.com/ipld/go-ipld-prime/schema\"\n")
	fmt.Fprintf(w, ")\n\n")
}

// EmitEntireType is a helper function calls all methods of TypeGenerator
// and streams all results into a single writer.
// (This implies two calls to EmitNode -- one for the type-level and one for the representation-level.)
func EmitEntireType(tg TypeGenerator, w io.Writer) {
	tg.EmitNativeType(w)
	tg.EmitNativeAccessors(w)
	tg.EmitNativeBuilder(w)
	tg.EmitNativeMaybe(w)
	EmitNode(tg, w)
	tg.EmitTypedNodeMethodType(w)
	tg.EmitTypedNodeMethodRepresentation(w)

	rng := tg.GetRepresentationNodeGen()
	if rng == nil { // FIXME: hack to save me from stubbing tons right now, remove when done
		return
	}
	EmitNode(rng, w)
}

// EmitNode is a helper function that calls all methods of NodeGenerator
// and streams all results into a single writer.
func EmitNode(ng NodeGenerator, w io.Writer) {
	ng.EmitNodeType(w)
	ng.EmitNodeTypeAssertions(w)
	ng.EmitNodeMethodKind(w)
	ng.EmitNodeMethodLookupByString(w)
	ng.EmitNodeMethodLookupByNode(w)
	ng.EmitNodeMethodLookupByIndex(w)
	ng.EmitNodeMethodLookupBySegment(w)
	ng.EmitNodeMethodMapIterator(w)
	ng.EmitNodeMethodListIterator(w)
	ng.EmitNodeMethodLength(w)
	ng.EmitNodeMethodIsAbsent(w)
	ng.EmitNodeMethodIsNull(w)
	ng.EmitNodeMethodAsBool(w)
	ng.EmitNodeMethodAsInt(w)
	ng.EmitNodeMethodAsFloat(w)
	ng.EmitNodeMethodAsString(w)
	ng.EmitNodeMethodAsBytes(w)
	ng.EmitNodeMethodAsLink(w)
	ng.EmitNodeMethodPrototype(w)

	ng.EmitNodePrototypeType(w)

	nbg := ng.GetNodeBuilderGenerator()
	if nbg == nil { // FIXME: hack to save me from stubbing tons right now, remove when done
		return
	}
	nbg.EmitNodeBuilderType(w)
	nbg.EmitNodeBuilderMethods(w)
	nbg.EmitNodeAssemblerType(w)
	nbg.EmitNodeAssemblerMethodBeginMap(w)
	nbg.EmitNodeAssemblerMethodBeginList(w)
	nbg.EmitNodeAssemblerMethodAssignNull(w)
	nbg.EmitNodeAssemblerMethodAssignBool(w)
	nbg.EmitNodeAssemblerMethodAssignInt(w)
	nbg.EmitNodeAssemblerMethodAssignFloat(w)
	nbg.EmitNodeAssemblerMethodAssignString(w)
	nbg.EmitNodeAssemblerMethodAssignBytes(w)
	nbg.EmitNodeAssemblerMethodAssignLink(w)
	nbg.EmitNodeAssemblerMethodAssignNode(w)
	nbg.EmitNodeAssemblerMethodPrototype(w)
	nbg.EmitNodeAssemblerOtherBits(w)
}

func EmitTypeTable(pkgName string, ts schema.TypeSystem, adjCfg *AdjunctCfg, w io.Writer) {
	// REVIEW: if "T__Repr" is how we want to expose this.  We could also put 'Repr' accessors on the type/prototype objects.
	// FUTURE: types and prototypes are proposed to be the same.  Some of this text pretends they already are, but work is needed on this.
	doTemplate(`
		// Type is a struct embeding a NodePrototype/Type for every Node implementation in this package.
		// One of its major uses is to start the construction of a value.
		// You can use it like this:
		//
		// 		`+pkgName+`.Type.YourTypeName.NewBuilder().BeginMap() //...
		//
		// and:
		//
		// 		`+pkgName+`.Type.OtherTypeName.NewBuilder().AssignString("x") // ...
		//
		var Type typeSlab

		type typeSlab struct {
			{{- range . }}
			{{ .Name }}       _{{ . | TypeSymbol }}__Prototype
			{{ .Name }}__Repr _{{ . | TypeSymbol }}__ReprPrototype
			{{- end}}
		}
	`, w, adjCfg, ts.GetTypes())
}
