//go:build ignore

package main

// based on https://github.com/ipld/go-ipld-prime-proto/blob/master/gen/main.go

import (
	"fmt"
	"os"

	"github.com/ipld/go-ipld-prime/schema"
	gengo "github.com/ipld/go-ipld-prime/schema/gen/go"
)

func main() {
	ts := schema.TypeSystem{}
	ts.Init()
	adjCfg := &gengo.AdjunctCfg{}

	pkgName := "dagpb"

	ts.Accumulate(schema.SpawnString("String"))
	ts.Accumulate(schema.SpawnInt("Int"))
	ts.Accumulate(schema.SpawnLink("Link"))
	ts.Accumulate(schema.SpawnBytes("Bytes"))

	/*
		type PBLink struct {
			Hash Link
			Name optional String
			Tsize optional Int
		}
	*/

	ts.Accumulate(schema.SpawnStruct("PBLink",
		[]schema.StructField{
			schema.SpawnStructField("Hash", "Link", false, false),
			schema.SpawnStructField("Name", "String", true, false),
			schema.SpawnStructField("Tsize", "Int", true, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnList("PBLinks", "PBLink", false))

	/*
		type PBNode struct {
			Links [PBLink]
			Data optional Bytes
		}
	*/

	ts.Accumulate(schema.SpawnStruct("PBNode",
		[]schema.StructField{
			schema.SpawnStructField("Links", "PBLinks", false, false),
			schema.SpawnStructField("Data", "Bytes", true, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))

	if errs := ts.ValidateGraph(); errs != nil {
		for _, err := range errs {
			fmt.Printf("- %s\n", err)
		}
		os.Exit(1)
	}

	gengo.Generate(".", pkgName, ts, adjCfg)
}
