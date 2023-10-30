//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/ipld/go-ipld-prime/node/bindnode"
	schemadmt "github.com/ipld/go-ipld-prime/schema/dmt"
)

func main() {
	f, err := os.Create("types.go")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(f, "package schemadmt\n\n")
	if err := bindnode.ProduceGoTypes(f, schemadmt.TypeSystem); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}
