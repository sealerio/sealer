package bindnode

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/node/mixins"
	"github.com/ipld/go-ipld-prime/schema"
)

// Assert that we implement all the interfaces as expected.
// Grouped by the interfaces to implement, roughly.
var (
	_ datamodel.NodePrototype = (*_prototype)(nil)
	_ schema.TypedPrototype   = (*_prototype)(nil)
	_ datamodel.NodePrototype = (*_prototypeRepr)(nil)

	_ datamodel.Node   = (*_node)(nil)
	_ schema.TypedNode = (*_node)(nil)
	_ datamodel.Node   = (*_nodeRepr)(nil)

	_ datamodel.Node     = (*_uintNode)(nil)
	_ schema.TypedNode   = (*_uintNode)(nil)
	_ datamodel.UintNode = (*_uintNode)(nil)
	_ datamodel.Node     = (*_uintNodeRepr)(nil)
	_ datamodel.UintNode = (*_uintNodeRepr)(nil)

	_ datamodel.NodeBuilder   = (*_builder)(nil)
	_ datamodel.NodeBuilder   = (*_builderRepr)(nil)
	_ datamodel.NodeAssembler = (*_assembler)(nil)
	_ datamodel.NodeAssembler = (*_assemblerRepr)(nil)
	_ datamodel.NodeAssembler = (*_errorAssembler)(nil)
	_ datamodel.NodeAssembler = (*_listpairsFieldAssemblerRepr)(nil)

	_ datamodel.MapAssembler = (*_structAssembler)(nil)
	_ datamodel.MapAssembler = (*_structAssemblerRepr)(nil)
	_ datamodel.MapIterator  = (*_structIterator)(nil)
	_ datamodel.MapIterator  = (*_structIteratorRepr)(nil)

	_ datamodel.ListAssembler = (*_listAssembler)(nil)
	_ datamodel.ListAssembler = (*_listAssemblerRepr)(nil)
	_ datamodel.ListAssembler = (*_listStructAssemblerRepr)(nil)
	_ datamodel.ListAssembler = (*_listpairsFieldListAssemblerRepr)(nil)
	_ datamodel.ListIterator  = (*_listIterator)(nil)
	_ datamodel.ListIterator  = (*_tupleIteratorRepr)(nil)
	_ datamodel.ListIterator  = (*_listpairsIteratorRepr)(nil)

	_ datamodel.MapAssembler = (*_unionAssembler)(nil)
	_ datamodel.MapAssembler = (*_unionAssemblerRepr)(nil)
	_ datamodel.MapIterator  = (*_unionIterator)(nil)
	_ datamodel.MapIterator  = (*_unionIteratorRepr)(nil)
)

type _prototype struct {
	cfg        config
	schemaType schema.Type
	goType     reflect.Type // non-pointer
}

func (w *_prototype) NewBuilder() datamodel.NodeBuilder {
	return &_builder{_assembler{
		cfg:        w.cfg,
		schemaType: w.schemaType,
		val:        reflect.New(w.goType).Elem(),
	}}
}

func (w *_prototype) Type() schema.Type {
	return w.schemaType
}

func (w *_prototype) Representation() datamodel.NodePrototype {
	return (*_prototypeRepr)(w)
}

type _node struct {
	cfg        config
	schemaType schema.Type

	val reflect.Value // non-pointer
}

// TODO: only expose TypedNode methods if the schema was explicit.
// type _typedNode struct {
// 	_node
// }

func newNode(cfg config, schemaType schema.Type, val reflect.Value) schema.TypedNode {
	if schemaType.TypeKind() == schema.TypeKind_Int && nonPtrVal(val).Kind() == reflect.Uint64 {
		// special case for uint64 values so we can handle the >int64 range
		// we give this treatment to all uint64s, regardless of current value
		// because we have no guarantees the value won't change underneath us
		return &_uintNode{
			cfg:        cfg,
			schemaType: schemaType,
			val:        val,
		}
	}
	return &_node{cfg, schemaType, val}
}

func (w *_node) Type() schema.Type {
	return w.schemaType
}

func (w *_node) Representation() datamodel.Node {
	return (*_nodeRepr)(w)
}

func (w *_node) Kind() datamodel.Kind {
	return actualKind(w.schemaType)
}

// matching schema level types to data model kinds, since our Node and Builder
// interfaces operate on kinds
func compatibleKind(schemaType schema.Type, kind datamodel.Kind) error {
	switch sch := schemaType.(type) {
	case *schema.TypeAny:
		return nil
	default:
		actual := actualKind(sch) // ActsLike data model
		if actual == kind {
			return nil
		}

		// Error
		methodName := ""
		if pc, _, _, ok := runtime.Caller(1); ok {
			if fn := runtime.FuncForPC(pc); fn != nil {
				methodName = fn.Name()
				// Go from "pkg/path.Type.Method" to just "Method".
				methodName = methodName[strings.LastIndexByte(methodName, '.')+1:]
			}
		}
		return datamodel.ErrWrongKind{
			TypeName:        schemaType.Name(),
			MethodName:      methodName,
			AppropriateKind: datamodel.KindSet{kind},
			ActualKind:      actual,
		}
	}
}

func actualKind(schemaType schema.Type) datamodel.Kind {
	return schemaType.TypeKind().ActsLike()
}

func nonPtrVal(val reflect.Value) reflect.Value {
	// TODO: support **T as well as *T?
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			// TODO: error in this case?
			return reflect.Value{}
		}
		val = val.Elem()
	}
	return val
}

func ptrVal(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr {
		return val
	}
	return val.Addr()
}

func nonPtrType(val reflect.Value) reflect.Type {
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

// where we need to cal Set(), ensure the Value we're setting is a pointer or
// not, depending on the field we're setting into.
func matchSettable(val interface{}, to reflect.Value) reflect.Value {
	setVal := nonPtrVal(reflect.ValueOf(val))
	if !setVal.Type().AssignableTo(to.Type()) && setVal.Type().ConvertibleTo(to.Type()) {
		setVal = setVal.Convert(to.Type())
	}
	return setVal
}

func (w *_node) LookupByString(key string) (datamodel.Node, error) {
	switch typ := w.schemaType.(type) {
	case *schema.TypeStruct:
		field := typ.Field(key)
		if field == nil {
			return nil, schema.ErrInvalidKey{
				TypeName: typ.Name(),
				Key:      basicnode.NewString(key),
			}
		}
		fval := nonPtrVal(w.val).FieldByName(fieldNameFromSchema(key))
		if !fval.IsValid() {
			return nil, fmt.Errorf("bindnode TODO: go-schema mismatch")
		}
		if field.IsOptional() {
			if fval.IsNil() {
				return datamodel.Absent, nil
			}
			if fval.Kind() == reflect.Ptr {
				fval = fval.Elem()
			}
		}
		if field.IsNullable() {
			if fval.IsNil() {
				return datamodel.Null, nil
			}
			if fval.Kind() == reflect.Ptr {
				fval = fval.Elem()
			}
		}
		if _, ok := field.Type().(*schema.TypeAny); ok {
			if customConverter := w.cfg.converterFor(fval); customConverter != nil {
				// field is an Any and we have a custom type converter for the type
				return customConverter.customToAny(ptrVal(fval).Interface())
			}
			// field is an Any, safely assume a Node in fval
			return nonPtrVal(fval).Interface().(datamodel.Node), nil
		}
		return newNode(w.cfg, field.Type(), fval), nil
	case *schema.TypeMap:
		// maps can only be structs with a Values map
		var kval reflect.Value
		valuesVal := nonPtrVal(w.val).FieldByName("Values")
		switch ktyp := typ.KeyType().(type) {
		case *schema.TypeString:
			// plain String keys, so safely use the map key as is
			kval = reflect.ValueOf(key)
		default:
			// key is something other than a string that we need to assemble via
			// the string representation form, use _assemblerRepr to reverse from
			// string to the type that indexes the map
			asm := &_assembler{
				cfg:        w.cfg,
				schemaType: ktyp,
				val:        reflect.New(valuesVal.Type().Key()).Elem(),
			}
			if err := (*_assemblerRepr)(asm).AssignString(key); err != nil {
				return nil, err
			}
			kval = asm.val
		}
		fval := valuesVal.MapIndex(kval)
		if !fval.IsValid() { // not found
			return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
		}
		// TODO: Error/panic if fval.IsNil() && !typ.ValueIsNullable()?
		// Otherwise we could have two non-equal Go values (nil map,
		// non-nil-but-empty map) which represent the exact same IPLD
		// node when the field is not nullable.
		if typ.ValueIsNullable() {
			if fval.IsNil() {
				return datamodel.Null, nil
			}
			fval = fval.Elem()
		}
		if _, ok := typ.ValueType().(*schema.TypeAny); ok {
			if customConverter := w.cfg.converterFor(fval); customConverter != nil {
				// value is an Any and we have a custom type converter for the type
				return customConverter.customToAny(ptrVal(fval).Interface())
			}
			// value is an Any, safely assume a Node in fval
			return nonPtrVal(fval).Interface().(datamodel.Node), nil
		}
		return newNode(w.cfg, typ.ValueType(), fval), nil
	case *schema.TypeUnion:
		// treat a union similar to a struct, but we have the member names more
		// easily accessible to match to 'key'
		var idx int
		var mtyp schema.Type
		for i, member := range typ.Members() {
			if member.Name() == key {
				idx = i
				mtyp = member
				break
			}
		}
		if mtyp == nil { // not found
			return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
		}
		// TODO: we could look up the right Go field straight away via idx.
		haveIdx, mval := unionMember(nonPtrVal(w.val))
		if haveIdx != idx { // mismatching type
			return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
		}
		return newNode(w.cfg, mtyp, mval), nil
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "LookupByString",
		AppropriateKind: datamodel.KindSet_JustMap,
		ActualKind:      w.Kind(),
	}
}

var invalidValue reflect.Value

// unionMember finds which union member is set in the corresponding Go struct.
func unionMember(val reflect.Value) (int, reflect.Value) {
	// The first non-nil field is a match.
	for i := 0; i < val.NumField(); i++ {
		elemVal := val.Field(i)
		if elemVal.Kind() != reflect.Ptr {
			panic("bindnode bug: found unexpected non-pointer in a union field")
		}
		if elemVal.IsNil() {
			continue
		}
		return i, elemVal.Elem()
	}
	return -1, invalidValue
}

func unionSetMember(val reflect.Value, memberIdx int, memberPtr reflect.Value) {
	// Reset the entire union struct to zero, to clear any non-nil pointers.
	val.Set(reflect.Zero(val.Type()))

	// Set the index pointer to the given value.
	val.Field(memberIdx).Set(memberPtr)
}

func (w *_node) LookupByIndex(idx int64) (datamodel.Node, error) {
	switch typ := w.schemaType.(type) {
	case *schema.TypeList:
		val := nonPtrVal(w.val)
		// we should be able assume that val is something we can Len() and Index()
		if idx < 0 || int(idx) >= val.Len() {
			return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfInt(idx)}
		}
		val = val.Index(int(idx))
		_, isAny := typ.ValueType().(*schema.TypeAny)
		if isAny {
			if customConverter := w.cfg.converterFor(val); customConverter != nil {
				// values are Any and we have a converter for this type that will give us
				// a datamodel.Node
				return customConverter.customToAny(ptrVal(val).Interface())
			}
		}
		if typ.ValueIsNullable() {
			if val.IsNil() {
				return datamodel.Null, nil
			}
			// nullable elements are assumed to be pointers
			val = val.Elem()
		}
		if isAny {
			// Any always yields a plain datamodel.Node
			return nonPtrVal(val).Interface().(datamodel.Node), nil
		}
		return newNode(w.cfg, typ.ValueType(), val), nil
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "LookupByIndex",
		AppropriateKind: datamodel.KindSet_JustList,
		ActualKind:      w.Kind(),
	}
}

func (w *_node) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	switch w.Kind() {
	case datamodel.Kind_Map:
		return w.LookupByString(seg.String())
	case datamodel.Kind_List:
		idx, err := seg.Index()
		if err != nil {
			return nil, err
		}
		return w.LookupByIndex(idx)
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "LookupBySegment",
		AppropriateKind: datamodel.KindSet_Recursive,
		ActualKind:      w.Kind(),
	}
}

func (w *_node) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	switch w.Kind() {
	case datamodel.Kind_Map:
		s, err := key.AsString()
		if err != nil {
			return nil, err
		}
		return w.LookupByString(s)
	case datamodel.Kind_List:
		i, err := key.AsInt()
		if err != nil {
			return nil, err
		}
		return w.LookupByIndex(i)
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "LookupByNode",
		AppropriateKind: datamodel.KindSet_Recursive,
		ActualKind:      w.Kind(),
	}
}

func (w *_node) MapIterator() datamodel.MapIterator {
	val := nonPtrVal(w.val)
	// structs, unions and maps can all iterate but they each have different
	// access semantics for the underlying type, so we need a different iterator
	// for each
	switch typ := w.schemaType.(type) {
	case *schema.TypeStruct:
		return &_structIterator{
			cfg:        w.cfg,
			schemaType: typ,
			fields:     typ.Fields(),
			val:        val,
		}
	case *schema.TypeUnion:
		return &_unionIterator{
			cfg:        w.cfg,
			schemaType: typ,
			members:    typ.Members(),
			val:        val,
		}
	case *schema.TypeMap:
		// we can assume a: struct{Keys []string, Values map[x]y}
		return &_mapIterator{
			cfg:        w.cfg,
			schemaType: typ,
			keysVal:    val.FieldByName("Keys"),
			valuesVal:  val.FieldByName("Values"),
		}
	}
	return nil
}

func (w *_node) ListIterator() datamodel.ListIterator {
	val := nonPtrVal(w.val)
	switch typ := w.schemaType.(type) {
	case *schema.TypeList:
		return &_listIterator{cfg: w.cfg, schemaType: typ, val: val}
	}
	return nil
}

func (w *_node) Length() int64 {
	val := nonPtrVal(w.val)
	switch w.Kind() {
	case datamodel.Kind_Map:
		switch typ := w.schemaType.(type) {
		case *schema.TypeStruct:
			return int64(len(typ.Fields()))
		case *schema.TypeUnion:
			return 1
		}
		return int64(val.FieldByName("Keys").Len())
	case datamodel.Kind_List:
		return int64(val.Len())
	}
	return -1
}

// TODO: better story around pointers and absent/null

func (w *_node) IsAbsent() bool {
	return false
}

func (w *_node) IsNull() bool {
	return false
}

// The AsX methods are matter of fetching the non-pointer form of the underlying
// value and returning the appropriate Go type. The user may have registered
// custom converters for the kind being converted, in which case the underlying
// type may not be the type we need, but the converter will supply it for us.

func (w *_node) AsBool() (bool, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Bool); err != nil {
		return false, err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns a bool
		return customConverter.customToBool(ptrVal(w.val).Interface())
	}
	return nonPtrVal(w.val).Bool(), nil
}

func (w *_node) AsInt() (int64, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Int); err != nil {
		return 0, err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns an int
		return customConverter.customToInt(ptrVal(w.val).Interface())
	}
	val := nonPtrVal(w.val)
	if kindUint[val.Kind()] {
		u := val.Uint()
		if u > math.MaxInt64 {
			return 0, fmt.Errorf("bindnode: integer overflow, %d is too large for an int64", u)
		}
		return int64(u), nil
	}
	return val.Int(), nil
}

func (w *_node) AsFloat() (float64, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Float); err != nil {
		return 0, err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns a float
		return customConverter.customToFloat(ptrVal(w.val).Interface())
	}
	return nonPtrVal(w.val).Float(), nil
}

func (w *_node) AsString() (string, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_String); err != nil {
		return "", err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns a string
		return customConverter.customToString(ptrVal(w.val).Interface())
	}
	return nonPtrVal(w.val).String(), nil
}

func (w *_node) AsBytes() ([]byte, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Bytes); err != nil {
		return nil, err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns a []byte
		return customConverter.customToBytes(ptrVal(w.val).Interface())
	}
	return nonPtrVal(w.val).Bytes(), nil
}

func (w *_node) AsLink() (datamodel.Link, error) {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Link); err != nil {
		return nil, err
	}
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns a cid.Cid
		cid, err := customConverter.customToLink(ptrVal(w.val).Interface())
		if err != nil {
			return nil, err
		}
		return cidlink.Link{Cid: cid}, nil
	}
	switch val := nonPtrVal(w.val).Interface().(type) {
	case datamodel.Link:
		return val, nil
	case cid.Cid:
		return cidlink.Link{Cid: val}, nil
	default:
		return nil, fmt.Errorf("bindnode: unexpected link type %T", val)
	}
}

func (w *_node) Prototype() datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: w.schemaType, goType: w.val.Type()}
}

type _builder struct {
	_assembler
}

func (w *_builder) Build() datamodel.Node {
	// TODO: should we panic if no Assign call was made, just like codegen?
	return newNode(w.cfg, w.schemaType, w.val)
}

func (w *_builder) Reset() {
	panic("bindnode TODO: Reset")
}

type _assembler struct {
	cfg        config
	schemaType schema.Type
	val        reflect.Value // non-pointer

	// finish is used as an optional post-assemble step.
	// For example, assigning to a kinded union uses a finish func
	// to set the right union member in the Go union struct,
	// which isn't known before the assemble has finished.
	finish func() error

	nullable bool // true if field or map value is nullable
}

// createNonPtrVal is used for Set() operations on the underlying value
func (w *_assembler) createNonPtrVal() reflect.Value {
	val := w.val
	// TODO: if val is not a pointer, we reuse its value.
	// If it is a pointer, we allocate a new one and replace it.
	// We should probably never reuse the existing value.

	// TODO: support **T as well as *T?
	if val.Kind() == reflect.Ptr {
		// TODO: Sometimes we call createNonPtrVal before an assignment actually
		// happens. Does that matter?
		// If it matters and we only want to modify the destination value on
		// success, then we should make use of the "finish" func.
		val.Set(reflect.New(val.Type().Elem()))
		val = val.Elem()
	}
	return val
}

func (w *_assembler) Representation() datamodel.NodeAssembler {
	return (*_assemblerRepr)(w)
}

// basicMapAssembler is for assembling basicnode values, it's only use is for
// Any fields that end up needing a BeginMap()
type basicMapAssembler struct {
	datamodel.MapAssembler

	builder   datamodel.NodeBuilder
	parent    *_assembler
	converter *converter
}

func (w *basicMapAssembler) Finish() error {
	if err := w.MapAssembler.Finish(); err != nil {
		return err
	}
	basicNode := w.builder.Build()
	if w.converter != nil {
		// we can assume an Any converter because basicMapAssembler is only for Any
		// the user has registered the ability to convert a datamodel.Node to the
		// underlying Go type which may not be a datamodel.Node
		typ, err := w.converter.customFromAny(basicNode)
		if err != nil {
			return err
		}
		w.parent.createNonPtrVal().Set(matchSettable(typ, reflect.ValueOf(basicNode)))
	} else {
		w.parent.createNonPtrVal().Set(reflect.ValueOf(basicNode))
	}
	if w.parent.finish != nil {
		if err := w.parent.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	switch typ := w.schemaType.(type) {
	case *schema.TypeAny:
		basicBuilder := basicnode.Prototype.Any.NewBuilder()
		mapAsm, err := basicBuilder.BeginMap(sizeHint)
		if err != nil {
			return nil, err
		}
		converter := w.cfg.converterFor(w.val)
		return &basicMapAssembler{MapAssembler: mapAsm, builder: basicBuilder, parent: w, converter: converter}, nil
	case *schema.TypeStruct:
		val := w.createNonPtrVal()
		// _structAssembler walks through the fields in order as the entries are
		// assembled, verifyCompatibility() should mean it's safe to assume that
		// they match the schema, but we need to keep track of the fields that are
		// set in case of premature Finish()
		doneFields := make([]bool, val.NumField())
		return &_structAssembler{
			cfg:        w.cfg,
			schemaType: typ,
			val:        val,
			doneFields: doneFields,
			finish:     w.finish,
		}, nil
	case *schema.TypeMap:
		// assume a struct{Keys []string, Values map[x]y} that we can fill with
		// _mapAssembler
		val := w.createNonPtrVal()
		keysVal := val.FieldByName("Keys")
		valuesVal := val.FieldByName("Values")
		if valuesVal.IsNil() {
			valuesVal.Set(reflect.MakeMap(valuesVal.Type()))
		}
		return &_mapAssembler{
			cfg:        w.cfg,
			schemaType: typ,
			keysVal:    keysVal,
			valuesVal:  valuesVal,
			finish:     w.finish,
		}, nil
	case *schema.TypeUnion:
		// we can use _unionAssembler to assemble a union as if it were a map with
		// a single entry
		val := w.createNonPtrVal()
		return &_unionAssembler{
			cfg:        w.cfg,
			schemaType: typ,
			val:        val,
			finish:     w.finish,
		}, nil
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "BeginMap",
		AppropriateKind: datamodel.KindSet_JustMap,
		ActualKind:      actualKind(w.schemaType),
	}
}

// basicListAssembler is for assembling basicnode values, it's only use is for
// Any fields that end up needing a BeginList()
type basicListAssembler struct {
	datamodel.ListAssembler

	builder   datamodel.NodeBuilder
	parent    *_assembler
	converter *converter
}

func (w *basicListAssembler) Finish() error {
	if err := w.ListAssembler.Finish(); err != nil {
		return err
	}
	basicNode := w.builder.Build()
	if w.converter != nil {
		// we can assume an Any converter because basicListAssembler is only for Any
		// the user has registered the ability to convert a datamodel.Node to the
		// underlying Go type which may not be a datamodel.Node
		typ, err := w.converter.customFromAny(basicNode)
		if err != nil {
			return err
		}
		w.parent.createNonPtrVal().Set(matchSettable(typ, reflect.ValueOf(basicNode)))
	} else {
		w.parent.createNonPtrVal().Set(reflect.ValueOf(basicNode))
	}
	if w.parent.finish != nil {
		if err := w.parent.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	switch typ := w.schemaType.(type) {
	case *schema.TypeAny:
		basicBuilder := basicnode.Prototype.Any.NewBuilder()
		listAsm, err := basicBuilder.BeginList(sizeHint)
		if err != nil {
			return nil, err
		}
		converter := w.cfg.converterFor(w.val)
		return &basicListAssembler{ListAssembler: listAsm, builder: basicBuilder, parent: w, converter: converter}, nil
	case *schema.TypeList:
		// we should be able to safely assume we're dealing with a Go slice here,
		// so _listAssembler can append to that
		val := w.createNonPtrVal()
		return &_listAssembler{
			cfg:        w.cfg,
			schemaType: typ,
			val:        val,
			finish:     w.finish,
		}, nil
	}
	return nil, datamodel.ErrWrongKind{
		TypeName:        w.schemaType.Name(),
		MethodName:      "BeginList",
		AppropriateKind: datamodel.KindSet_JustList,
		ActualKind:      actualKind(w.schemaType),
	}
}

func (w *_assembler) AssignNull() error {
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter := w.cfg.converterFor(w.val); customConverter != nil && isAny {
		// an Any field that is being assigned a Null, we pass the Null directly to
		// the converter, regardless of whether this field is nullable or not
		typ, err := customConverter.customFromAny(datamodel.Null)
		if err != nil {
			return err
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if !w.nullable {
			return datamodel.ErrWrongKind{
				TypeName:   w.schemaType.Name(),
				MethodName: "AssignNull",
				// TODO
			}
		}
		// set the zero value for the underlying type as a stand-in for Null
		w.val.Set(reflect.Zero(w.val.Type()))
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignBool(b bool) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Bool); err != nil {
		return err
	}
	customConverter := w.cfg.converterFor(w.val)
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter != nil {
		var typ interface{}
		var err error
		if isAny {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			if typ, err = customConverter.customFromAny(basicnode.NewBool(b)); err != nil {
				return err
			}
		} else {
			// field is a Bool, but the user has registered a converter from a bool to
			// whatever the underlying Go type is
			if typ, err = customConverter.customFromBool(b); err != nil {
				return err
			}
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if isAny {
			// Any means the Go type must receive a datamodel.Node
			w.createNonPtrVal().Set(reflect.ValueOf(basicnode.NewBool(b)))
		} else {
			w.createNonPtrVal().SetBool(b)
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) assignUInt(uin datamodel.UintNode) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Int); err != nil {
		return err
	}
	_, isAny := w.schemaType.(*schema.TypeAny)
	// TODO: customConverter for uint??
	if isAny {
		// Any means the Go type must receive a datamodel.Node
		w.createNonPtrVal().Set(reflect.ValueOf(uin))
	} else {
		i, err := uin.AsUint()
		if err != nil {
			return err
		}
		if kindUint[w.val.Kind()] {
			w.createNonPtrVal().SetUint(i)
		} else {
			// TODO: check for overflow
			w.createNonPtrVal().SetInt(int64(i))
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignInt(i int64) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Int); err != nil {
		return err
	}
	// TODO: check for overflow
	customConverter := w.cfg.converterFor(w.val)
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter != nil {
		var typ interface{}
		var err error
		if isAny {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			if typ, err = customConverter.customFromAny(basicnode.NewInt(i)); err != nil {
				return err
			}
		} else {
			// field is an Int, but the user has registered a converter from an int to
			// whatever the underlying Go type is
			if typ, err = customConverter.customFromInt(i); err != nil {
				return err
			}
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if isAny {
			// Any means the Go type must receive a datamodel.Node
			w.createNonPtrVal().Set(reflect.ValueOf(basicnode.NewInt(i)))
		} else if kindUint[w.val.Kind()] {
			if i < 0 {
				// TODO: write a test
				return fmt.Errorf("bindnode: cannot assign negative integer to %s", w.val.Type())
			}
			w.createNonPtrVal().SetUint(uint64(i))
		} else {
			w.createNonPtrVal().SetInt(i)
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignFloat(f float64) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Float); err != nil {
		return err
	}
	customConverter := w.cfg.converterFor(w.val)
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter != nil {
		var typ interface{}
		var err error
		if isAny {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			if typ, err = customConverter.customFromAny(basicnode.NewFloat(f)); err != nil {
				return err
			}
		} else {
			// field is a Float, but the user has registered a converter from a float
			// to whatever the underlying Go type is
			if typ, err = customConverter.customFromFloat(f); err != nil {
				return err
			}
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if isAny {
			// Any means the Go type must receive a datamodel.Node
			w.createNonPtrVal().Set(reflect.ValueOf(basicnode.NewFloat(f)))
		} else {
			w.createNonPtrVal().SetFloat(f)
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignString(s string) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_String); err != nil {
		return err
	}
	customConverter := w.cfg.converterFor(w.val)
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter != nil {
		var typ interface{}
		var err error
		if isAny {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			if typ, err = customConverter.customFromAny(basicnode.NewString(s)); err != nil {
				return err
			}
		} else {
			// field is a String, but the user has registered a converter from a
			// string to whatever the underlying Go type is
			if typ, err = customConverter.customFromString(s); err != nil {
				return err
			}
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if isAny {
			// Any means the Go type must receive a datamodel.Node
			w.createNonPtrVal().Set(reflect.ValueOf(basicnode.NewString(s)))
		} else {
			w.createNonPtrVal().SetString(s)
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignBytes(p []byte) error {
	if err := compatibleKind(w.schemaType, datamodel.Kind_Bytes); err != nil {
		return err
	}
	customConverter := w.cfg.converterFor(w.val)
	_, isAny := w.schemaType.(*schema.TypeAny)
	if customConverter != nil {
		var typ interface{}
		var err error
		if isAny {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			if typ, err = customConverter.customFromAny(basicnode.NewBytes(p)); err != nil {
				return err
			}
		} else {
			// field is a Bytes, but the user has registered a converter from a []byte
			// to whatever the underlying Go type is
			if typ, err = customConverter.customFromBytes(p); err != nil {
				return err
			}
		}
		w.createNonPtrVal().Set(matchSettable(typ, w.val))
	} else {
		if isAny {
			// Any means the Go type must receive a datamodel.Node
			w.createNonPtrVal().Set(reflect.ValueOf(basicnode.NewBytes(p)))
		} else {
			w.createNonPtrVal().SetBytes(p)
		}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignLink(link datamodel.Link) error {
	val := w.createNonPtrVal()
	// TODO: newVal.Type() panics if link==nil; add a test and fix.
	customConverter := w.cfg.converterFor(w.val)
	if _, ok := w.schemaType.(*schema.TypeAny); ok {
		if customConverter != nil {
			// field is an Any, so the converter will be an Any converter that wants
			// a datamodel.Node to convert to whatever the underlying Go type is
			typ, err := customConverter.customFromAny(basicnode.NewLink(link))
			if err != nil {
				return err
			}
			w.createNonPtrVal().Set(matchSettable(typ, w.val))
		} else {
			// Any means the Go type must receive a datamodel.Node
			val.Set(reflect.ValueOf(basicnode.NewLink(link)))
		}
	} else if customConverter != nil {
		if cl, ok := link.(cidlink.Link); ok {
			// field is a Link, but the user has registered a converter from a cid.Cid
			// to whatever the underlying Go type is
			typ, err := customConverter.customFromLink(cl.Cid)
			if err != nil {
				return err
			}
			w.createNonPtrVal().Set(matchSettable(typ, w.val))
		} else {
			return fmt.Errorf("bindnode: custom converter can only receive a cidlink.Link through AssignLink")
		}
	} else if newVal := reflect.ValueOf(link); newVal.Type().AssignableTo(val.Type()) {
		// Directly assignable.
		val.Set(newVal)
	} else if newVal.Type() == goTypeCidLink && goTypeCid.AssignableTo(val.Type()) {
		// Unbox a cidlink.Link to assign to a go-cid.Cid value.
		newVal = newVal.FieldByName("Cid")
		val.Set(newVal)
	} else if actual := actualKind(w.schemaType); actual != datamodel.Kind_Link {
		// We're assigning a Link to a schema type that isn't a Link.
		return datamodel.ErrWrongKind{
			TypeName:        w.schemaType.Name(),
			MethodName:      "AssignLink",
			AppropriateKind: datamodel.KindSet_JustLink,
			ActualKind:      actualKind(w.schemaType),
		}
	} else {
		// The schema type is a Link, but we somehow can't assign to the Go value.
		// Almost certainly a bug; we should have verified for compatibility upfront.
		return fmt.Errorf("bindnode bug: AssignLink with %s argument can't be used on Go type %s",
			newVal.Type(), val.Type())
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_assembler) AssignNode(node datamodel.Node) error {
	// TODO: does this ever trigger?
	// newVal := reflect.ValueOf(node)
	// if newVal.Type().AssignableTo(w.val.Type()) {
	// 	w.val.Set(newVal)
	// 	return nil
	// }
	if uintNode, ok := node.(datamodel.UintNode); ok {
		return w.assignUInt(uintNode)
	}
	return datamodel.Copy(node, w)
}

func (w *_assembler) Prototype() datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: w.schemaType, goType: w.val.Type()}
}

// _structAssembler is used for Struct assembling via BeginMap()
type _structAssembler struct {
	// TODO: embed _assembler?

	cfg config

	schemaType *schema.TypeStruct
	val        reflect.Value // non-pointer
	finish     func() error

	// TODO: more state checks

	// TODO: Consider if we could do this in a cheaper way,
	// such as looking at the reflect.Value directly.
	// If not, at least avoid an extra alloc.
	doneFields []bool

	// TODO: optimize for structs

	curKey _assembler

	nextIndex int // only used by repr.go
}

func (w *_structAssembler) AssembleKey() datamodel.NodeAssembler {
	w.curKey = _assembler{
		cfg:        w.cfg,
		schemaType: schemaTypeString,
		val:        reflect.New(goTypeString).Elem(),
	}
	return &w.curKey
}

func (w *_structAssembler) AssembleValue() datamodel.NodeAssembler {
	// TODO: optimize this to do one lookup by name
	name := w.curKey.val.String()
	field := w.schemaType.Field(name)
	if field == nil {
		// TODO: should've been raised when the key was submitted instead.
		// TODO: should make well-typed errors for this.
		return _errorAssembler{fmt.Errorf("bindnode TODO: invalid key: %q is not a field in type %s", name, w.schemaType.Name())}
		// panic(schema.ErrInvalidKey{
		// 	TypeName: w.schemaType.Name(),
		// 	Key:      basicnode.NewString(name),
		// })
	}
	ftyp, ok := w.val.Type().FieldByName(fieldNameFromSchema(name))
	if !ok {
		// It is unfortunate this is not detected proactively earlier during bind.
		return _errorAssembler{fmt.Errorf("schema type %q has field %q, we expect go struct to have field %q", w.schemaType.Name(), field.Name(), fieldNameFromSchema(name))}
	}
	if len(ftyp.Index) > 1 {
		return _errorAssembler{fmt.Errorf("bindnode TODO: embedded fields")}
	}
	w.doneFields[ftyp.Index[0]] = true
	fval := w.val.FieldByIndex(ftyp.Index)
	if field.IsOptional() {
		if fval.Kind() == reflect.Ptr {
			// ptrVal = new(T); val = *ptrVal
			fval.Set(reflect.New(fval.Type().Elem()))
			fval = fval.Elem()
		} else {
			// val = *new(T)
			fval.Set(reflect.New(fval.Type()).Elem())
		}
	}
	// TODO: reuse same assembler for perf?
	return &_assembler{
		cfg:        w.cfg,
		schemaType: field.Type(),
		val:        fval,
		nullable:   field.IsNullable(),
	}
}

func (w *_structAssembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
	if err := w.AssembleKey().AssignString(k); err != nil {
		return nil, err
	}
	am := w.AssembleValue()
	return am, nil
}

func (w *_structAssembler) Finish() error {
	fields := w.schemaType.Fields()
	var missing []string
	for i, field := range fields {
		if !field.IsOptional() && !w.doneFields[i] {
			missing = append(missing, field.Name())
		}
	}
	if len(missing) > 0 {
		return schema.ErrMissingRequiredField{Missing: missing}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_structAssembler) KeyPrototype() datamodel.NodePrototype {
	// TODO: if the user provided their own schema with their own typesystem,
	// the schemaTypeString here may be using the wrong typesystem.
	return &_prototype{cfg: w.cfg, schemaType: schemaTypeString, goType: goTypeString}
}

func (w *_structAssembler) ValuePrototype(k string) datamodel.NodePrototype {
	panic("bindnode TODO: struct ValuePrototype")
}

type _errorAssembler struct {
	err error
}

func (w _errorAssembler) BeginMap(int64) (datamodel.MapAssembler, error)   { return nil, w.err }
func (w _errorAssembler) BeginList(int64) (datamodel.ListAssembler, error) { return nil, w.err }
func (w _errorAssembler) AssignNull() error                                { return w.err }
func (w _errorAssembler) AssignBool(bool) error                            { return w.err }
func (w _errorAssembler) AssignInt(int64) error                            { return w.err }
func (w _errorAssembler) AssignFloat(float64) error                        { return w.err }
func (w _errorAssembler) AssignString(string) error                        { return w.err }
func (w _errorAssembler) AssignBytes([]byte) error                         { return w.err }
func (w _errorAssembler) AssignLink(datamodel.Link) error                  { return w.err }
func (w _errorAssembler) AssignNode(datamodel.Node) error                  { return w.err }
func (w _errorAssembler) Prototype() datamodel.NodePrototype               { return nil }

// used for Maps which we can assume are of type: struct{Keys []string, Values map[x]y},
// where we have Keys in keysVal and Values in valuesVal
type _mapAssembler struct {
	cfg        config
	schemaType *schema.TypeMap
	keysVal    reflect.Value // non-pointer
	valuesVal  reflect.Value // non-pointer
	finish     func() error

	// TODO: more state checks

	curKey _assembler
}

func (w *_mapAssembler) AssembleKey() datamodel.NodeAssembler {
	w.curKey = _assembler{
		cfg:        w.cfg,
		schemaType: w.schemaType.KeyType(),
		val:        reflect.New(w.valuesVal.Type().Key()).Elem(),
	}
	return &w.curKey
}

func (w *_mapAssembler) AssembleValue() datamodel.NodeAssembler {
	kval := w.curKey.val
	val := reflect.New(w.valuesVal.Type().Elem()).Elem()
	finish := func() error {
		// TODO: check for duplicates in keysVal
		w.keysVal.Set(reflect.Append(w.keysVal, kval))

		w.valuesVal.SetMapIndex(kval, val)
		return nil
	}
	return &_assembler{
		cfg:        w.cfg,
		schemaType: w.schemaType.ValueType(),
		val:        val,
		nullable:   w.schemaType.ValueIsNullable(),
		finish:     finish,
	}
}

func (w *_mapAssembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
	if err := w.AssembleKey().AssignString(k); err != nil {
		return nil, err
	}
	am := w.AssembleValue()
	return am, nil
}

func (w *_mapAssembler) Finish() error {
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_mapAssembler) KeyPrototype() datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: w.schemaType.KeyType(), goType: w.valuesVal.Type().Key()}
}

func (w *_mapAssembler) ValuePrototype(k string) datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: w.schemaType.ValueType(), goType: w.valuesVal.Type().Elem()}
}

// _listAssembler is for operating directly on slices, which we have in val
type _listAssembler struct {
	cfg        config
	schemaType *schema.TypeList
	val        reflect.Value // non-pointer
	finish     func() error
}

func (w *_listAssembler) AssembleValue() datamodel.NodeAssembler {
	goType := w.val.Type().Elem()
	// TODO: use a finish func to append
	w.val.Set(reflect.Append(w.val, reflect.New(goType).Elem()))
	return &_assembler{
		cfg:        w.cfg,
		schemaType: w.schemaType.ValueType(),
		val:        w.val.Index(w.val.Len() - 1),
		nullable:   w.schemaType.ValueIsNullable(),
	}
}

func (w *_listAssembler) Finish() error {
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_listAssembler) ValuePrototype(idx int64) datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: w.schemaType.ValueType(), goType: w.val.Type().Elem()}
}

// when assembling as a Map but we anticipate a single value, which we need to
// look up in the union members
type _unionAssembler struct {
	cfg        config
	schemaType *schema.TypeUnion
	val        reflect.Value // non-pointer
	finish     func() error

	// TODO: more state checks

	curKey _assembler
}

func (w *_unionAssembler) AssembleKey() datamodel.NodeAssembler {
	w.curKey = _assembler{
		cfg:        w.cfg,
		schemaType: schemaTypeString,
		val:        reflect.New(goTypeString).Elem(),
	}
	return &w.curKey
}

func (w *_unionAssembler) AssembleValue() datamodel.NodeAssembler {
	name := w.curKey.val.String()
	var idx int
	var mtyp schema.Type
	for i, member := range w.schemaType.Members() {
		if member.Name() == name {
			idx = i
			mtyp = member
			break
		}
	}
	if mtyp == nil {
		return _errorAssembler{fmt.Errorf("bindnode TODO: missing member %s in %s", name, w.schemaType.Name())}
		// return nil, datamodel.ErrInvalidKey{
		// 	TypeName: w.schemaType.Name(),
		// 	Key:      basicnode.NewString(name),
		// }
	}

	goType := w.val.Field(idx).Type().Elem()
	valPtr := reflect.New(goType)
	finish := func() error {
		unionSetMember(w.val, idx, valPtr)
		return nil
	}
	return &_assembler{
		cfg:        w.cfg,
		schemaType: mtyp,
		val:        valPtr.Elem(),
		finish:     finish,
	}
}

func (w *_unionAssembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
	if err := w.AssembleKey().AssignString(k); err != nil {
		return nil, err
	}
	am := w.AssembleValue()
	return am, nil
}

func (w *_unionAssembler) Finish() error {
	// TODO(rvagg): I think this might allow setting multiple members of the union
	// we need a test for this.
	haveIdx, _ := unionMember(w.val)
	if haveIdx < 0 {
		return schema.ErrNotUnionStructure{TypeName: w.schemaType.Name(), Detail: "a union must have exactly one entry"}
	}
	if w.finish != nil {
		if err := w.finish(); err != nil {
			return err
		}
	}
	return nil
}

func (w *_unionAssembler) KeyPrototype() datamodel.NodePrototype {
	return &_prototype{cfg: w.cfg, schemaType: schemaTypeString, goType: goTypeString}
}

func (w *_unionAssembler) ValuePrototype(k string) datamodel.NodePrototype {
	panic("bindnode TODO: union ValuePrototype")
}

// _structIterator is for iterating over Struct types which operate over Go
// structs. The iteration order is dictated by Go field declaration order which
// should match the schema for this type.
type _structIterator struct {
	// TODO: support embedded fields?
	cfg config

	schemaType *schema.TypeStruct
	fields     []schema.StructField
	val        reflect.Value // non-pointer
	nextIndex  int

	// these are only used in repr.go
	reprEnd int
}

func (w *_structIterator) Next() (key, value datamodel.Node, _ error) {
	if w.Done() {
		return nil, nil, datamodel.ErrIteratorOverread{}
	}
	field := w.fields[w.nextIndex]
	val := w.val.Field(w.nextIndex)
	w.nextIndex++
	key = basicnode.NewString(field.Name())
	if field.IsOptional() {
		if val.IsNil() {
			return key, datamodel.Absent, nil
		}
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
	}
	_, isAny := field.Type().(*schema.TypeAny)
	if isAny {
		if customConverter := w.cfg.converterFor(val); customConverter != nil {
			// field is an Any and we have an Any converter which takes the underlying
			// struct field value and returns a datamodel.Node
			v, err := customConverter.customToAny(ptrVal(val).Interface())
			if err != nil {
				return nil, nil, err
			}
			return key, v, nil
		}
	}
	if field.IsNullable() {
		if val.IsNil() {
			return key, datamodel.Null, nil
		}
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
	}
	if isAny {
		// field holds a datamodel.Node
		return key, nonPtrVal(val).Interface().(datamodel.Node), nil
	}
	return key, newNode(w.cfg, field.Type(), val), nil
}

func (w *_structIterator) Done() bool {
	return w.nextIndex >= len(w.fields)
}

// _mapIterator is for iterating over a struct{Keys []string, Values map[x]y},
// where we have the Keys in keysVal and Values in valuesVal
type _mapIterator struct {
	cfg        config
	schemaType *schema.TypeMap
	keysVal    reflect.Value // non-pointer
	valuesVal  reflect.Value // non-pointer
	nextIndex  int
}

func (w *_mapIterator) Next() (key, value datamodel.Node, _ error) {
	if w.Done() {
		return nil, nil, datamodel.ErrIteratorOverread{}
	}
	goKey := w.keysVal.Index(w.nextIndex)
	val := w.valuesVal.MapIndex(goKey)
	w.nextIndex++

	key = newNode(w.cfg, w.schemaType.KeyType(), goKey)
	_, isAny := w.schemaType.ValueType().(*schema.TypeAny)
	if isAny {
		if customConverter := w.cfg.converterFor(val); customConverter != nil {
			// values of this map are Any and we have an Any converter which takes the
			// underlying map value and returns a datamodel.Node

			// TODO(rvagg): can't call ptrVal on a map value that's not a pointer
			// so only map[string]*foo will work for the Values map and an Any
			// converter. Should we check in infer.go?
			val, err := customConverter.customToAny(ptrVal(val).Interface())
			return key, val, err
		}
	}
	if w.schemaType.ValueIsNullable() {
		if val.IsNil() {
			return key, datamodel.Null, nil
		}
		val = val.Elem() // nullable entries are pointers
	}
	if isAny {
		// Values holds datamodel.Nodes
		return key, nonPtrVal(val).Interface().(datamodel.Node), nil
	}
	return key, newNode(w.cfg, w.schemaType.ValueType(), val), nil
}

func (w *_mapIterator) Done() bool {
	return w.nextIndex >= w.keysVal.Len()
}

// _listIterator is for iterating over slices, which is held in val
type _listIterator struct {
	cfg        config
	schemaType *schema.TypeList
	val        reflect.Value // non-pointer
	nextIndex  int
}

func (w *_listIterator) Next() (index int64, value datamodel.Node, _ error) {
	if w.Done() {
		return 0, nil, datamodel.ErrIteratorOverread{}
	}
	idx := int64(w.nextIndex)
	val := w.val.Index(w.nextIndex)
	w.nextIndex++
	if w.schemaType.ValueIsNullable() {
		if val.IsNil() {
			return idx, datamodel.Null, nil
		}
		val = val.Elem() // nullable values are pointers
	}
	if _, ok := w.schemaType.ValueType().(*schema.TypeAny); ok {
		if customConverter := w.cfg.converterFor(val); customConverter != nil {
			// values are Any and we have an Any converter which can take whatever
			// the underlying Go type in this slice is and return a datamodel.Node
			val, err := customConverter.customToAny(ptrVal(val).Interface())
			return idx, val, err
		}
		// values are Any, assume that they are datamodel.Nodes
		return idx, nonPtrVal(val).Interface().(datamodel.Node), nil
	}
	return idx, newNode(w.cfg, w.schemaType.ValueType(), val), nil
}

func (w *_listIterator) Done() bool {
	return w.nextIndex >= w.val.Len()
}

type _unionIterator struct {
	// TODO: support embedded fields?
	cfg        config
	schemaType *schema.TypeUnion
	members    []schema.Type
	val        reflect.Value // non-pointer

	done bool
}

func (w *_unionIterator) Next() (key, value datamodel.Node, _ error) {
	// we can only call this once for a union since a union can only have one
	// entry even though it behaves like a Map
	if w.Done() {
		return nil, nil, datamodel.ErrIteratorOverread{}
	}
	w.done = true

	haveIdx, mval := unionMember(w.val)
	if haveIdx < 0 {
		return nil, nil, fmt.Errorf("bindnode: union %s has no member", w.val.Type())
	}
	mtyp := w.members[haveIdx]

	node := newNode(w.cfg, mtyp, mval)
	key = basicnode.NewString(mtyp.Name())
	return key, node, nil
}

func (w *_unionIterator) Done() bool {
	return w.done
}

// --- uint64 special case handling

type _uintNode struct {
	cfg        config
	schemaType schema.Type

	val reflect.Value // non-pointer
}

func (tu *_uintNode) Type() schema.Type {
	return tu.schemaType
}
func (tu *_uintNode) Representation() datamodel.Node {
	return (*_uintNodeRepr)(tu)
}
func (_uintNode) Kind() datamodel.Kind {
	return datamodel.Kind_Int
}
func (_uintNode) LookupByString(string) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByString("")
}
func (_uintNode) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByNode(nil)
}
func (_uintNode) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByIndex(0)
}
func (_uintNode) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupBySegment(seg)
}
func (_uintNode) MapIterator() datamodel.MapIterator {
	return nil
}
func (_uintNode) ListIterator() datamodel.ListIterator {
	return nil
}
func (_uintNode) Length() int64 {
	return -1
}
func (_uintNode) IsAbsent() bool {
	return false
}
func (_uintNode) IsNull() bool {
	return false
}
func (_uintNode) AsBool() (bool, error) {
	return mixins.Int{TypeName: "int"}.AsBool()
}
func (tu *_uintNode) AsInt() (int64, error) {
	return (*_uintNodeRepr)(tu).AsInt()
}
func (tu *_uintNode) AsUint() (uint64, error) {
	return (*_uintNodeRepr)(tu).AsUint()
}
func (_uintNode) AsFloat() (float64, error) {
	return mixins.Int{TypeName: "int"}.AsFloat()
}
func (_uintNode) AsString() (string, error) {
	return mixins.Int{TypeName: "int"}.AsString()
}
func (_uintNode) AsBytes() ([]byte, error) {
	return mixins.Int{TypeName: "int"}.AsBytes()
}
func (_uintNode) AsLink() (datamodel.Link, error) {
	return mixins.Int{TypeName: "int"}.AsLink()
}
func (_uintNode) Prototype() datamodel.NodePrototype {
	return basicnode.Prototype__Int{}
}

// we need this for _uintNode#Representation() so we don't return a TypeNode
type _uintNodeRepr _uintNode

func (_uintNodeRepr) Kind() datamodel.Kind {
	return datamodel.Kind_Int
}
func (_uintNodeRepr) LookupByString(string) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByString("")
}
func (_uintNodeRepr) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByNode(nil)
}
func (_uintNodeRepr) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupByIndex(0)
}
func (_uintNodeRepr) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.Int{TypeName: "int"}.LookupBySegment(seg)
}
func (_uintNodeRepr) MapIterator() datamodel.MapIterator {
	return nil
}
func (_uintNodeRepr) ListIterator() datamodel.ListIterator {
	return nil
}
func (_uintNodeRepr) Length() int64 {
	return -1
}
func (_uintNodeRepr) IsAbsent() bool {
	return false
}
func (_uintNodeRepr) IsNull() bool {
	return false
}
func (_uintNodeRepr) AsBool() (bool, error) {
	return mixins.Int{TypeName: "int"}.AsBool()
}
func (tu *_uintNodeRepr) AsInt() (int64, error) {
	if err := compatibleKind(tu.schemaType, datamodel.Kind_Int); err != nil {
		return 0, err
	}
	if customConverter := tu.cfg.converterFor(tu.val); customConverter != nil {
		// user has registered a converter that takes the underlying type and returns an int
		return customConverter.customToInt(ptrVal(tu.val).Interface())
	}
	val := nonPtrVal(tu.val)
	// we can assume it's a uint64 at this point
	u := val.Uint()
	if u > math.MaxInt64 {
		return 0, fmt.Errorf("bindnode: integer overflow, %d is too large for an int64", u)
	}
	return int64(u), nil
}
func (tu *_uintNodeRepr) AsUint() (uint64, error) {
	if err := compatibleKind(tu.schemaType, datamodel.Kind_Int); err != nil {
		return 0, err
	}
	// TODO(rvagg): do we want a converter option for uint values? do we combine it
	// with int converters?
	// we can assume it's a uint64 at this point
	return nonPtrVal(tu.val).Uint(), nil
}
func (_uintNodeRepr) AsFloat() (float64, error) {
	return mixins.Int{TypeName: "int"}.AsFloat()
}
func (_uintNodeRepr) AsString() (string, error) {
	return mixins.Int{TypeName: "int"}.AsString()
}
func (_uintNodeRepr) AsBytes() ([]byte, error) {
	return mixins.Int{TypeName: "int"}.AsBytes()
}
func (_uintNodeRepr) AsLink() (datamodel.Link, error) {
	return mixins.Int{TypeName: "int"}.AsLink()
}
func (_uintNodeRepr) Prototype() datamodel.NodePrototype {
	return basicnode.Prototype__Int{}
}
