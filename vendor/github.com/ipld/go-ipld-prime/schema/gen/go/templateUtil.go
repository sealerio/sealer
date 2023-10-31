package gengo

import (
	"io"
	"strings"
	"text/template"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/testutil"
)

func doTemplate(tmplstr string, w io.Writer, adjCfg *AdjunctCfg, data interface{}) {
	tmpl := template.Must(template.New("").
		Funcs(template.FuncMap{

			// These methods are used for symbol munging and appear constantly, so they need to be short.
			//  (You could also get at them through `.AdjCfg`, but going direct saves some screen real estate.)
			"TypeSymbol":       adjCfg.TypeSymbol,
			"FieldSymbolLower": adjCfg.FieldSymbolLower,
			"FieldSymbolUpper": adjCfg.FieldSymbolUpper,
			"MaybeUsesPtr":     adjCfg.MaybeUsesPtr,
			"Comments":         adjCfg.Comments,

			// The whole AdjunctConfig can be accessed.
			//  Access methods like UnionMemlayout through this, as e.g. `.AdjCfg.UnionMemlayout`.
			"AdjCfg": func() *AdjunctCfg { return adjCfg },

			// "dot" is a dummy value that's equal to the original `.` expression, but stays there.
			//  Use this if you're inside a range or other feature that shifted the dot and you want the original.
			//  (This may seem silly, but empirically, I found myself writing a dummy line to store the value of dot before endering a range clause >20 times; that's plenty.)
			"dot": func() interface{} { return data },

			"KindPrim": func(k datamodel.Kind) string {
				switch k {
				case datamodel.Kind_Map:
					panic("this isn't useful for non-scalars")
				case datamodel.Kind_List:
					panic("this isn't useful for non-scalars")
				case datamodel.Kind_Null:
					panic("this isn't useful for null")
				case datamodel.Kind_Bool:
					return "bool"
				case datamodel.Kind_Int:
					return "int64"
				case datamodel.Kind_Float:
					return "float64"
				case datamodel.Kind_String:
					return "string"
				case datamodel.Kind_Bytes:
					return "[]byte"
				case datamodel.Kind_Link:
					return "datamodel.Link"
				default:
					panic("invalid enumeration value!")
				}
			},
			"Kind": func(s string) datamodel.Kind {
				switch s {
				case "map":
					return datamodel.Kind_Map
				case "list":
					return datamodel.Kind_List
				case "null":
					return datamodel.Kind_Null
				case "bool":
					return datamodel.Kind_Bool
				case "int":
					return datamodel.Kind_Int
				case "float":
					return datamodel.Kind_Float
				case "string":
					return datamodel.Kind_String
				case "bytes":
					return datamodel.Kind_Bytes
				case "link":
					return datamodel.Kind_Link
				default:
					panic("invalid enumeration value!")
				}
			},
			"KindSymbol": func(k datamodel.Kind) string {
				switch k {
				case datamodel.Kind_Map:
					return "datamodel.Kind_Map"
				case datamodel.Kind_List:
					return "datamodel.Kind_List"
				case datamodel.Kind_Null:
					return "datamodel.Kind_Null"
				case datamodel.Kind_Bool:
					return "datamodel.Kind_Bool"
				case datamodel.Kind_Int:
					return "datamodel.Kind_Int"
				case datamodel.Kind_Float:
					return "datamodel.Kind_Float"
				case datamodel.Kind_String:
					return "datamodel.Kind_String"
				case datamodel.Kind_Bytes:
					return "datamodel.Kind_Bytes"
				case datamodel.Kind_Link:
					return "datamodel.Kind_Link"
				default:
					panic("invalid enumeration value!")
				}
			},
			"add":   func(a, b int) int { return a + b },
			"title": func(s string) string { return strings.Title(s) }, //lint:ignore SA1019 cases.Title doesn't work for this
		}).
		Parse(testutil.Dedent(tmplstr)))
	if err := tmpl.Execute(w, data); err != nil {
		panic(err)
	}
}

// We really need to do some more composable stuff around here.
// Generators should probably be carrying down their own doTemplate methods that curry customizations.
// E.g., map generators would benefit hugely from being able to make a clause for "entTypeStrung", "mTypeStrung", etc.
//
// Open question: how exactly?  Should some of this stuff should be composed by:
//   - composing template fragments;
//   - amending the funcmap;
//   - computing the whole result and injecting it as a string;
//   - ... combinations of the above?
// Adding to the complexity of the question is that sometimes we want to be
//  doing composition inside the output (e.g. DRY by functions in the result,
//   rather than by DRY'ing the templates).
// Best practice to make this evolve nicely is not at all obvious to this author.
//
