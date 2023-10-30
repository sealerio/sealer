package gengo

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/schema"
)

// This entire file is placeholder-quality implementations.
//
// The AdjunctCfg struct should be replaced with an IPLD Schema-specified thing!
// The values in the unionMemlayout field should be an enum;
// etcetera!

type FieldTuple struct {
	TypeName  schema.TypeName
	FieldName string
}

type AdjunctCfg struct {
	typeSymbolOverrides       map[schema.TypeName]string
	FieldSymbolLowerOverrides map[FieldTuple]string
	fieldSymbolUpperOverrides map[FieldTuple]string
	maybeUsesPtr              map[schema.TypeName]bool   // absent uses a heuristic
	CfgUnionMemlayout         map[schema.TypeName]string // "embedAll"|"interface"; maybe more options later, unclear for now.

	// ... some of these fields have sprouted messy name prefixes so they don't collide with their matching method names.
	//  this structure has reached the critical threshhold where it due to be cleaned up and taken seriously.

	// note: PkgName doesn't appear in here, because it's...
	//  not adjunct data.  it's a generation invocation parameter.
	//   ... this might not hold up in the future though.
	//    There are unanswered questions about how (also, tbf, *if*) we'll handle generation of multiple packages which use each other's types.
}

// TypeSymbol returns the symbol for a type;
// by default, it's the same string as its name in the schema,
// but it can be overriden.
//
// This is the base, unembellished symbol.
// It's frequently augmented:
// prefixing an underscore to make it unexported;
// suffixing "__Something" to make the name of a supporting type;
// etc.
// (Most such augmentations are not configurable.)
func (cfg *AdjunctCfg) TypeSymbol(t schema.Type) string {
	if x, ok := cfg.typeSymbolOverrides[t.Name()]; ok {
		return x
	}
	return string(t.Name()) // presumed already upper
}

func (cfg *AdjunctCfg) FieldSymbolLower(f schema.StructField) string {
	if x, ok := cfg.FieldSymbolLowerOverrides[FieldTuple{f.Parent().Name(), f.Name()}]; ok {
		return x
	}
	return f.Name() // presumed already lower
}

func (cfg *AdjunctCfg) FieldSymbolUpper(f schema.StructField) string {
	if x, ok := cfg.fieldSymbolUpperOverrides[FieldTuple{f.Type().Name(), f.Name()}]; ok {
		return x
	}
	return strings.Title(f.Name()) //lint:ignore SA1019 cases.Title doesn't work for this
}

// Comments returns a bool for whether comments should be included in gen output or not.
func (cfg *AdjunctCfg) Comments() bool {
	return true // FUTURE: okay, maybe this should be configurable :)
}

func (cfg *AdjunctCfg) MaybeUsesPtr(t schema.Type) bool {
	if x, ok := cfg.maybeUsesPtr[t.Name()]; ok {
		return x
	}

	// As a simple heuristic,
	// check how large the Go representation of this type will be.
	// If it weighs little, we estimate that a pointer is not worthwhile,
	// as storing the data directly will barely take more memory.
	// Plus, the resulting code will be shorter and have fewer branches.
	return sizeOfSchemaType(t) > sizeSmallEnoughForInlining
}

var (
	// The cutoff for "weighs little" is any size up to this number.
	// It's hasn't been measured with any benchmarks or stats just yet.
	// It's possible that, with those, it might increase in the future.
	// Intuitively, any type 4x the size of a pointer is fine to inline.
	// Adding a pointer will already add 1x overhead, anyway.
	sizeSmallEnoughForInlining = 4 * reflect.TypeOf(new(int)).Size()

	sizeOfTypeKind [128]uintptr
)

func init() {
	// Uncomment for debugging.
	// fmt.Fprintf(os.Stderr, "sizeOf(small): %d (4x pointer size)\n", sizeSmallEnoughForInlining)

	// Get the basic node sizes via basicnode.
	for _, tk := range []struct {
		typeKind  schema.TypeKind
		prototype datamodel.NodePrototype
	}{
		{schema.TypeKind_Bool, basicnode.Prototype.Bool},
		{schema.TypeKind_Int, basicnode.Prototype.Int},
		{schema.TypeKind_Float, basicnode.Prototype.Float},
		{schema.TypeKind_String, basicnode.Prototype.String},
		{schema.TypeKind_Bytes, basicnode.Prototype.Bytes},
		{schema.TypeKind_List, basicnode.Prototype.List},
		{schema.TypeKind_Map, basicnode.Prototype.Map},
		{schema.TypeKind_Link, basicnode.Prototype.Link},
	} {
		nb := tk.prototype.NewBuilder()
		switch tk.typeKind {
		case schema.TypeKind_List:
			am, err := nb.BeginList(0)
			if err != nil {
				panic(err)
			}
			if err := am.Finish(); err != nil {
				panic(err)
			}
		case schema.TypeKind_Map:
			am, err := nb.BeginMap(0)
			if err != nil {
				panic(err)
			}
			if err := am.Finish(); err != nil {
				panic(err)
			}
		}
		// Note that the Node interface has a pointer underneath,
		// so we use Elem to reach the underlying type.
		size := reflect.TypeOf(nb.Build()).Elem().Size()
		sizeOfTypeKind[tk.typeKind] = size

		// Uncomment for debugging.
		// fmt.Fprintf(os.Stderr, "sizeOf(%s): %d\n", tk.typeKind, size)
	}
}

// sizeOfSchemaType returns the size of a schema type,
// relative to the size of a pointer in native Go.
//
// For example, TypeInt and TypeMap returns 1, but TypeList returns 3, as a
// slice in Go has a pointer and two integers for length and capacity.
// Any basic type smaller than a pointer, such as TypeBool, returns 1.
func sizeOfSchemaType(t schema.Type) uintptr {
	kind := t.TypeKind()

	// If this TypeKind is represented by the basicnode package,
	// we statically know its size and we can return here.
	if size := sizeOfTypeKind[kind]; size > 0 {
		return size
	}

	// TODO: handle typekinds like structs, unions, etc.
	// For now, return a large size to fall back to using a pointer.
	return 100 * sizeSmallEnoughForInlining
}

// UnionMemlayout returns a plain string at present;
// there's a case-switch in the templates that processes it.
// We validate that it's a known string when this method is called.
// This should probably be improved in type-safety,
// and validated more aggressively up front when adjcfg is loaded.
func (cfg *AdjunctCfg) UnionMemlayout(t schema.Type) string {
	if t.TypeKind() != schema.TypeKind_Union {
		panic(fmt.Errorf("%s is not a union", t.Name()))
	}
	v, ok := cfg.CfgUnionMemlayout[t.Name()]
	if !ok {
		return "embedAll"
	}
	switch v {
	case "embedAll", "interface":
		return v
	default:
		panic(fmt.Errorf("invalid config: unionMemlayout values must be either \"embedAll\" or \"interface\", not %q", v))
	}
}
