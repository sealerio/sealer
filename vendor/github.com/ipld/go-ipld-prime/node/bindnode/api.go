// Package bindnode provides a datamodel.Node implementation via Go reflection.
//
// This package is EXPERIMENTAL; its behavior and API might change as it's still
// in development.
package bindnode

import (
	"reflect"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

// Prototype implements a schema.TypedPrototype given a Go pointer type and an
// IPLD schema type. Note that the result is also a datamodel.NodePrototype.
//
// If both the Go type and schema type are supplied, it is assumed that they are
// compatible with one another.
//
// If either the Go type or schema type are nil, we infer the missing type from
// the other provided type. For example, we can infer an unnamed Go struct type
// for a schema struct type, and we can infer a schema Int type for a Go int64
// type. The inferring logic is still a work in progress and subject to change.
// At this time, inferring IPLD Unions and Enums from Go types is not supported.
//
// When supplying a non-nil ptrType, Prototype only obtains the Go pointer type
// from it, so its underlying value will typically be nil. For example:
//
//	proto := bindnode.Prototype((*goType)(nil), schemaType)
func Prototype(ptrType interface{}, schemaType schema.Type, options ...Option) schema.TypedPrototype {
	if ptrType == nil && schemaType == nil {
		panic("bindnode: either ptrType or schemaType must not be nil")
	}

	cfg := applyOptions(options...)

	// TODO: if both are supplied, verify that they are compatible

	var goType reflect.Type
	if ptrType == nil {
		goType = inferGoType(schemaType, make(map[schema.TypeName]inferredStatus), 0)
	} else {
		goPtrType := reflect.TypeOf(ptrType)
		if goPtrType.Kind() != reflect.Ptr {
			panic("bindnode: ptrType must be a pointer")
		}
		goType = goPtrType.Elem()
		if goType.Kind() == reflect.Ptr {
			panic("bindnode: ptrType must not be a pointer to a pointer")
		}

		if schemaType == nil {
			schemaType = inferSchema(goType, 0)
		} else {
			verifyCompatibility(cfg, make(map[seenEntry]bool), goType, schemaType)
		}
	}

	return &_prototype{cfg: cfg, schemaType: schemaType, goType: goType}
}

type converter struct {
	kind schema.TypeKind

	customFromBool func(bool) (interface{}, error)
	customToBool   func(interface{}) (bool, error)

	customFromInt func(int64) (interface{}, error)
	customToInt   func(interface{}) (int64, error)

	customFromFloat func(float64) (interface{}, error)
	customToFloat   func(interface{}) (float64, error)

	customFromString func(string) (interface{}, error)
	customToString   func(interface{}) (string, error)

	customFromBytes func([]byte) (interface{}, error)
	customToBytes   func(interface{}) ([]byte, error)

	customFromLink func(cid.Cid) (interface{}, error)
	customToLink   func(interface{}) (cid.Cid, error)

	customFromAny func(datamodel.Node) (interface{}, error)
	customToAny   func(interface{}) (datamodel.Node, error)
}

type config map[reflect.Type]*converter

// this mainly exists to short-circuit the nonPtrType() call; the `Type()` variant
// exists for completeness
func (c config) converterFor(val reflect.Value) *converter {
	if len(c) == 0 {
		return nil
	}
	return c[nonPtrType(val)]
}

func (c config) converterForType(typ reflect.Type) *converter {
	if len(c) == 0 {
		return nil
	}
	return c[typ]
}

// Option is able to apply custom options to the bindnode API
type Option func(config)

// TypedBoolConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func(bool) (interface{}, error)
// and toFunc is of the form: func(interface{}) (bool, error)
// where interface{} is a pointer form of the type we are converting.
//
// TypedBoolConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedBoolConverter(ptrVal interface{}, from func(bool) (interface{}, error), to func(interface{}) (bool, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:           schema.TypeKind_Bool,
		customFromBool: from,
		customToBool:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedIntConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func(int64) (interface{}, error)
// and toFunc is of the form: func(interface{}) (int64, error)
// where interface{} is a pointer form of the type we are converting.
//
// TypedIntConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedIntConverter(ptrVal interface{}, from func(int64) (interface{}, error), to func(interface{}) (int64, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:          schema.TypeKind_Int,
		customFromInt: from,
		customToInt:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedFloatConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func(float64) (interface{}, error)
// and toFunc is of the form: func(interface{}) (float64, error)
// where interface{} is a pointer form of the type we are converting.
//
// TypedFloatConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedFloatConverter(ptrVal interface{}, from func(float64) (interface{}, error), to func(interface{}) (float64, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:            schema.TypeKind_Float,
		customFromFloat: from,
		customToFloat:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedStringConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func(string) (interface{}, error)
// and toFunc is of the form: func(interface{}) (string, error)
// where interface{} is a pointer form of the type we are converting.
//
// TypedStringConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedStringConverter(ptrVal interface{}, from func(string) (interface{}, error), to func(interface{}) (string, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:             schema.TypeKind_String,
		customFromString: from,
		customToString:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedBytesConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func([]byte) (interface{}, error)
// and toFunc is of the form: func(interface{}) ([]byte, error)
// where interface{} is a pointer form of the type we are converting.
//
// TypedBytesConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedBytesConverter(ptrVal interface{}, from func([]byte) (interface{}, error), to func(interface{}) ([]byte, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:            schema.TypeKind_Bytes,
		customFromBytes: from,
		customToBytes:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedLinkConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func([]byte) (interface{}, error)
// and toFunc is of the form: func(interface{}) ([]byte, error)
// where interface{} is a pointer form of the type we are converting.
//
// Beware that this API is only compatible with cidlink.Link types in the data
// model and may result in errors if attempting to convert from other
// datamodel.Link types.
//
// TypedLinkConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedLinkConverter(ptrVal interface{}, from func(cid.Cid) (interface{}, error), to func(interface{}) (cid.Cid, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:           schema.TypeKind_Link,
		customFromLink: from,
		customToLink:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

// TypedAnyConverter adds custom converter functions for a particular
// type as identified by a pointer in the first argument.
// The fromFunc is of the form: func(datamodel.Node) (interface{}, error)
// and toFunc is of the form: func(interface{}) (datamodel.Node, error)
// where interface{} is a pointer form of the type we are converting.
//
// This method should be able to deal with all forms of Any and return an error
// if the expected data forms don't match the expected.
//
// TypedAnyConverter is an EXPERIMENTAL API and may be removed or
// changed in a future release.
func TypedAnyConverter(ptrVal interface{}, from func(datamodel.Node) (interface{}, error), to func(interface{}) (datamodel.Node, error)) Option {
	customType := nonPtrType(reflect.ValueOf(ptrVal))
	converter := &converter{
		kind:          schema.TypeKind_Any,
		customFromAny: from,
		customToAny:   to,
	}
	return func(cfg config) {
		cfg[customType] = converter
	}
}

func applyOptions(opt ...Option) config {
	if len(opt) == 0 {
		// no need to allocate, we access it via converterFor and converterForType
		// which are safe for nil maps
		return nil
	}
	cfg := make(map[reflect.Type]*converter)
	for _, o := range opt {
		o(cfg)
	}
	return cfg
}

// Wrap implements a schema.TypedNode given a non-nil pointer to a Go value and an
// IPLD schema type. Note that the result is also a datamodel.Node.
//
// Wrap is meant to be used when one already has a Go value with data.
// As such, ptrVal must not be nil.
//
// Similar to Prototype, if schemaType is non-nil it is assumed to be compatible
// with the Go type, and otherwise it's inferred from the Go type.
func Wrap(ptrVal interface{}, schemaType schema.Type, options ...Option) schema.TypedNode {
	if ptrVal == nil {
		panic("bindnode: ptrVal must not be nil")
	}
	goPtrVal := reflect.ValueOf(ptrVal)
	if goPtrVal.Kind() != reflect.Ptr {
		panic("bindnode: ptrVal must be a pointer")
	}
	if goPtrVal.IsNil() {
		// Note that this can happen if ptrVal was a typed nil.
		panic("bindnode: ptrVal must not be nil")
	}
	cfg := applyOptions(options...)
	goVal := goPtrVal.Elem()
	if goVal.Kind() == reflect.Ptr {
		panic("bindnode: ptrVal must not be a pointer to a pointer")
	}
	if schemaType == nil {
		schemaType = inferSchema(goVal.Type(), 0)
	} else {
		// TODO(rvagg): explore ways to make this skippable by caching in the schema.Type
		// passed in to this function; e.g. if you call Prototype(), then you've gone through
		// this already, then calling .Type() on that could return a bindnode version of
		// schema.Type that has the config cached and can be assumed to have been checked or
		// inferred.
		verifyCompatibility(cfg, make(map[seenEntry]bool), goVal.Type(), schemaType)
	}
	return newNode(cfg, schemaType, goVal)
}

// TODO: consider making our own Node interface, like:
//
// type WrappedNode interface {
//     datamodel.Node
//     Unwrap() (ptrVal interface)
// }
//
// Pros: API is easier to understand, harder to mix up with other datamodel.Nodes.
// Cons: One usually only has a datamodel.Node, and type assertions can be weird.

// Unwrap takes a datamodel.Node implemented by Prototype or Wrap,
// and returns a pointer to the inner Go value.
//
// Unwrap returns nil if the node isn't implemented by this package.
func Unwrap(node datamodel.Node) (ptrVal interface{}) {
	var val reflect.Value
	switch node := node.(type) {
	case *_node:
		val = node.val
	case *_nodeRepr:
		val = node.val
	default:
		return nil
	}
	if val.Kind() == reflect.Ptr {
		panic("bindnode: didn't expect val to be a pointer")
	}
	return val.Addr().Interface()
}
