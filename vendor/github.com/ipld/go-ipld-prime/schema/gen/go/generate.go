package gengo

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/ipld/go-ipld-prime/schema"
)

// Generate takes a typesystem and the adjunct config for codegen,
// and emits generated code in the given path with the given package name.
//
// All of the files produced will match the pattern "ipldsch.*.gen.go".
func Generate(pth string, pkgName string, ts schema.TypeSystem, adjCfg *AdjunctCfg) {
	// Emit fixed bits.
	withFile(filepath.Join(pth, "ipldsch_minima.go"), func(f io.Writer) {
		EmitInternalEnums(pkgName, f)
	})

	externs, err := getExternTypes(pth)
	if err != nil {
		// Consider warning that duplication may be present due to inability to parse destination.
		externs = make(map[string]struct{})
	}

	// Local helper function for applying generation logic to each type.
	//  We will end up doing this more than once because in this layout, more than one file contains part of the story for each type.
	applyToEachType := func(fn func(tg TypeGenerator, w io.Writer), f io.Writer) {
		// Sort the type names so we have a determinisic order; this affects output consistency.
		//  Any stable order would do, but we don't presently have one, so a sort is necessary.
		types := ts.GetTypes()
		keys := make(sortableTypeNames, 0, len(types))
		for tn := range types {
			if _, exists := externs[tn]; !exists {
				keys = append(keys, tn)
			}
		}
		sort.Sort(keys)
		for _, tn := range keys {
			switch t2 := types[tn].(type) {
			case *schema.TypeBool:
				fn(NewBoolReprBoolGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeInt:
				fn(NewIntReprIntGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeFloat:
				fn(NewFloatReprFloatGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeString:
				fn(NewStringReprStringGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeBytes:
				fn(NewBytesReprBytesGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeLink:
				fn(NewLinkReprLinkGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeStruct:
				switch t2.RepresentationStrategy().(type) {
				case schema.StructRepresentation_Map:
					fn(NewStructReprMapGenerator(pkgName, t2, adjCfg), f)
				case schema.StructRepresentation_Tuple:
					fn(NewStructReprTupleGenerator(pkgName, t2, adjCfg), f)
				case schema.StructRepresentation_Stringjoin:
					fn(NewStructReprStringjoinGenerator(pkgName, t2, adjCfg), f)
				default:
					panic("unrecognized struct representation strategy")
				}
			case *schema.TypeMap:
				fn(NewMapReprMapGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeList:
				fn(NewListReprListGenerator(pkgName, t2, adjCfg), f)
			case *schema.TypeUnion:
				switch t2.RepresentationStrategy().(type) {
				case schema.UnionRepresentation_Keyed:
					fn(NewUnionReprKeyedGenerator(pkgName, t2, adjCfg), f)
				case schema.UnionRepresentation_Kinded:
					fn(NewUnionReprKindedGenerator(pkgName, t2, adjCfg), f)
				case schema.UnionRepresentation_Stringprefix:
					fn(NewUnionReprStringprefixGenerator(pkgName, t2, adjCfg), f)
				default:
					panic("unrecognized union representation strategy")
				}
			default:
				panic(fmt.Sprintf("add more type switches here :), failed at type %s", tn))
			}
		}
	}

	// Emit a file with the type table, and the golang type defns for each type.
	withFile(filepath.Join(pth, "ipldsch_types.go"), func(f io.Writer) {
		// Emit headers, import statements, etc.
		fmt.Fprintf(f, "package %s\n\n", pkgName)
		fmt.Fprintf(f, doNotEditComment+"\n\n")
		fmt.Fprintf(f, "import (\n")
		fmt.Fprintf(f, "\t\"github.com/ipld/go-ipld-prime/datamodel\"\n") // referenced for links
		fmt.Fprintf(f, ")\n")
		fmt.Fprintf(f, "var _ datamodel.Node = nil // suppress errors when this dependency is not referenced\n")

		// Emit the type table.
		EmitTypeTable(pkgName, ts, adjCfg, f)

		// Emit the type defns matching the schema types.
		fmt.Fprintf(f, "\n// --- type definitions follow ---\n\n")
		applyToEachType(func(tg TypeGenerator, w io.Writer) {
			tg.EmitNativeType(w)
			fmt.Fprintf(f, "\n")
		}, f)

	})

	// Emit a file with all the Node/NodeBuilder/NodeAssembler boilerplate.
	//  Also includes typedefs for representation-level data.
	//  Also includes the MaybeT boilerplate.
	withFile(filepath.Join(pth, "ipldsch_satisfaction.go"), func(f io.Writer) {
		// Emit headers, import statements, etc.
		fmt.Fprintf(f, "package %s\n\n", pkgName)
		fmt.Fprintf(f, doNotEditComment+"\n\n")
		fmt.Fprintf(f, "import (\n")
		fmt.Fprintf(f, "\t\"github.com/ipld/go-ipld-prime/datamodel\"\n")   // referenced everywhere.
		fmt.Fprintf(f, "\t\"github.com/ipld/go-ipld-prime/node/mixins\"\n") // referenced by node implementation guts.
		fmt.Fprintf(f, "\t\"github.com/ipld/go-ipld-prime/schema\"\n")      // referenced by maybes (and surprisingly little else).
		fmt.Fprintf(f, ")\n\n")

		// For each type, we'll emit... everything except the native type, really.
		applyToEachType(func(tg TypeGenerator, w io.Writer) {
			tg.EmitNativeAccessors(w)
			tg.EmitNativeBuilder(w)
			tg.EmitNativeMaybe(w)
			EmitNode(tg, w)
			tg.EmitTypedNodeMethodType(w)
			tg.EmitTypedNodeMethodRepresentation(w)

			nrg := tg.GetRepresentationNodeGen()
			EmitNode(nrg, w)

			fmt.Fprintf(f, "\n")
		}, f)
	})
}

func withFile(filename string, fn func(io.Writer)) {
	// Don't write directly to the file, as that many write syscalls can be
	// expensive. Moreover, they can have a knock-on effect on daemons
	// watching for file changes. gopls can easily eat CPU for many seconds
	// just handling tens of thousands of file writes, for example.
	//
	// To alleviate both of those problems, write to a buffer first, and
	// then write the resulting bytes to disk in a single go.
	// A buffer is slightly better than bufio.Writer, as it gets us a bit
	// more atomicity via the single write.
	buf := new(bytes.Buffer)
	fn(buf)

	src := buf.Bytes()
	// Format the source before writing, just like gofmt would.
	// This also prevents us from writing invalid syntax to disk.
	src, err := format.Source(src)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(filename, src, 0666); err != nil {
		panic(err)
	}
}

type sortableTypeNames []schema.TypeName

func (a sortableTypeNames) Len() int           { return len(a) }
func (a sortableTypeNames) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortableTypeNames) Less(i, j int) bool { return a[i] < a[j] }
