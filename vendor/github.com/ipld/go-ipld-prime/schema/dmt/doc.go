/*
Package schema/dmt contains types and functions for dealing with the data model form of IPLD Schemas.

(DMT is short for "data model tree" -- see https://ipld.io/glossary/#dmt .)

As with anything that's IPLD data model, this data can be serialized or deserialized into a wide variety of codecs.

To contrast this package with some of its neighbors and with some various formats for the data this package describes:
Schemas also have a DSL (a domain-specific language -- something that's meant to look nice, and be easy for humans to read and write),
which are parsed by the `schema/dsl` package, and produce a DMT form (defined by and handled by this package).
Schemas also have a compiled form, which is the in-memory structure that this library uses when working with them;
this compiled form differs from the DMT because it can use pointers (and that includes cyclic pointers, which is something the DMT form cannot contain).
We use the DMT form (this package) to produce the compiled form (which is the `schema` package).

Creating a Compiled schema either flows from DSL(text)->`schema/dsl`->`schema/dmt`->`schema`,
or just (some codec, e.g. JSON or CBOR or etc)->`schema/dmt`->`schema`.

The `dmt.Schema` type describes the data found at the root of an IPLD Schema document.
The `Compile` function turns such data into a `schema.TypeSystem` that is ready to be used.
The `dmt.Prototype.Schema` value is a NodePrototype that can be used to handle IPLD Schemas in DMT form as regular IPLD Nodes.

Typically this package is imported aliased as "schemadmt",
since "dmt" is a fairly generic term in the IPLD ecosystem
(see https://ipld.io/glossary/#dmt ).

Many types in this package lack documentation directly on the type;
generally, these are structs that match the IPLD schema-schema,
and so you can find descriptions of them in documentation for the schema-schema.
*/
package schemadmt
