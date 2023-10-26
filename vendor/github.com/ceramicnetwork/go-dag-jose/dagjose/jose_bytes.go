package dagjose

import (
	"encoding/base64"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/mixins"
	"github.com/ipld/go-ipld-prime/schema"
)

// Base64Url matches the IPLD Schema type "Base64Url".  It has string kind.
type Base64Url = *_Base64Url
type _Base64Url struct{ x string }

type _Base64Url__Maybe struct {
	m schema.Maybe
	v _Base64Url
}
type MaybeBase64Url = *_Base64Url__Maybe

func (m MaybeBase64Url) IsNull() bool {
	return m.m == schema.Maybe_Null
}
func (m MaybeBase64Url) IsAbsent() bool {
	return m.m == schema.Maybe_Absent
}
func (m MaybeBase64Url) Exists() bool {
	return m.m == schema.Maybe_Value
}
func (m MaybeBase64Url) AsNode() datamodel.Node {
	switch m.m {
	case schema.Maybe_Absent:
		return datamodel.Absent
	case schema.Maybe_Null:
		return datamodel.Null
	case schema.Maybe_Value:
		return &m.v
	default:
		panic("unreachable")
	}
}
func (m MaybeBase64Url) Must() Base64Url {
	if !m.Exists() {
		panic("unbox of a maybe rejected")
	}
	return &m.v
}

var _ datamodel.Node = (Base64Url)(&_Base64Url{})
var _ schema.TypedNode = (Base64Url)(&_Base64Url{})

func (Base64Url) Kind() datamodel.Kind {
	return datamodel.Kind_String
}
func (Base64Url) LookupByString(string) (datamodel.Node, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.LookupByString("")
}
func (Base64Url) LookupByNode(datamodel.Node) (datamodel.Node, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.LookupByNode(nil)
}
func (Base64Url) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.LookupByIndex(0)
}
func (Base64Url) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.LookupBySegment(seg)
}
func (Base64Url) MapIterator() datamodel.MapIterator {
	return nil
}
func (Base64Url) ListIterator() datamodel.ListIterator {
	return nil
}
func (Base64Url) Length() int64 {
	return -1
}
func (Base64Url) IsAbsent() bool {
	return false
}
func (Base64Url) IsNull() bool {
	return false
}
func (Base64Url) AsBool() (bool, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.AsBool()
}
func (Base64Url) AsInt() (int64, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.AsInt()
}
func (Base64Url) AsFloat() (float64, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.AsFloat()
}
func (n Base64Url) AsString() (string, error) {
	return encodeBase64Url([]byte(n.x)), nil
}
func (n Base64Url) AsBytes() ([]byte, error) {
	return []byte(n.x), nil
}
func (Base64Url) AsLink() (datamodel.Link, error) {
	return mixins.String{TypeName: "dagjose.Base64Url"}.AsLink()
}
func (Base64Url) Prototype() datamodel.NodePrototype {
	return _Base64Url__Prototype{}
}

type _Base64Url__Prototype struct{}

func (_Base64Url__Prototype) NewBuilder() datamodel.NodeBuilder {
	var nb _Base64Url__Builder
	nb.Reset()
	return &nb
}

type _Base64Url__Builder struct {
	_Base64Url__Assembler
}

func (nb *_Base64Url__Builder) Build() datamodel.Node {
	if *nb.m != schema.Maybe_Value {
		panic("invalid state: cannot call Build on an assembler that's not finished")
	}
	return nb.w
}
func (nb *_Base64Url__Builder) Reset() {
	var w _Base64Url
	var m schema.Maybe
	*nb = _Base64Url__Builder{_Base64Url__Assembler{w: &w, m: &m}}
}

type _Base64Url__Assembler struct {
	w *_Base64Url
	m *schema.Maybe
}

func (na *_Base64Url__Assembler) reset() {}
func (_Base64Url__Assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.BeginMap(0)
}
func (_Base64Url__Assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.BeginList(0)
}
func (na *_Base64Url__Assembler) AssignNull() error {
	switch *na.m {
	case allowNull:
		*na.m = schema.Maybe_Null
		return nil
	case schema.Maybe_Absent:
		return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.AssignNull()
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	panic("unreachable")
}
func (_Base64Url__Assembler) AssignBool(bool) error {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.AssignBool(false)
}
func (_Base64Url__Assembler) AssignInt(int64) error {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.AssignInt(0)
}
func (_Base64Url__Assembler) AssignFloat(float64) error {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.AssignFloat(0)
}
func (na *_Base64Url__Assembler) AssignString(v string) error {
	switch *na.m {
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	if decodedBytes, err := decodeBase64Url(v); err != nil {
		return err
	} else {
		na.w.x = string(decodedBytes)
		*na.m = schema.Maybe_Value
		return nil
	}
}
func (na *_Base64Url__Assembler) AssignBytes(v []byte) error {
	switch *na.m {
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	na.w.x = string(v)
	*na.m = schema.Maybe_Value
	return nil
}
func (_Base64Url__Assembler) AssignLink(datamodel.Link) error {
	return mixins.StringAssembler{TypeName: "dagjose.Base64Url"}.AssignLink(nil)
}
func (na *_Base64Url__Assembler) AssignNode(v datamodel.Node) error {
	if v.IsNull() {
		return na.AssignNull()
	}
	if v2, ok := v.(*_Base64Url); ok {
		switch *na.m {
		case schema.Maybe_Value, schema.Maybe_Null:
			panic("invalid state: cannot assign into assembler that's already finished")
		}
		*na.w = *v2
		*na.m = schema.Maybe_Value
		return nil
	}
	if v2, err := v.AsString(); err != nil {
		if e, wrongKind := err.(datamodel.ErrWrongKind); wrongKind && (e.ActualKind == datamodel.Kind_Bytes) {
			if v2, err := v.AsBytes(); err != nil {
				return err
			} else {
				return na.AssignBytes(v2)
			}
		}
		return err
	} else {
		return na.AssignString(v2)
	}
}
func (_Base64Url__Assembler) Prototype() datamodel.NodePrototype {
	return _Base64Url__Prototype{}
}
func (Base64Url) Type() schema.Type {
	return nil
}
func (n Base64Url) Representation() datamodel.Node {
	return (*_Base64Url__Repr)(n)
}

type _Base64Url__Repr = _Base64Url

var _ datamodel.Node = &_Base64Url__Repr{}

type _Base64Url__ReprPrototype = _Base64Url__Prototype
type _Base64Url__ReprAssembler = _Base64Url__Assembler

func (_Base64Url__Prototype) Link(n Base64Url) (Link, error) {
	c, err := cid.Cast([]byte(n.x))
	if err != nil {
		return nil, err
	}
	return &_Link{cidlink.Link{Cid: c}}, nil
}

// Raw matches the IPLD Schema type "Raw".  It has bytes kind.
type Raw = *_Raw
type _Raw struct{ x []byte }

type _Raw__Maybe struct {
	m schema.Maybe
	v _Raw
}
type MaybeRaw = *_Raw__Maybe

func (m MaybeRaw) IsNull() bool {
	return m.m == schema.Maybe_Null
}
func (m MaybeRaw) IsAbsent() bool {
	return m.m == schema.Maybe_Absent
}
func (m MaybeRaw) Exists() bool {
	return m.m == schema.Maybe_Value
}
func (m MaybeRaw) AsNode() datamodel.Node {
	switch m.m {
	case schema.Maybe_Absent:
		return datamodel.Absent
	case schema.Maybe_Null:
		return datamodel.Null
	case schema.Maybe_Value:
		return &m.v
	default:
		panic("unreachable")
	}
}
func (m MaybeRaw) Must() Raw {
	if !m.Exists() {
		panic("unbox of a maybe rejected")
	}
	return &m.v
}

var _ datamodel.Node = (Raw)(&_Raw{})
var _ schema.TypedNode = (Raw)(&_Raw{})

func (Raw) Kind() datamodel.Kind {
	return datamodel.Kind_Bytes
}
func (Raw) LookupByString(string) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.LookupByString("")
}
func (Raw) LookupByNode(datamodel.Node) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.LookupByNode(nil)
}
func (Raw) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.LookupByIndex(0)
}
func (Raw) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.LookupBySegment(seg)
}
func (Raw) MapIterator() datamodel.MapIterator {
	return nil
}
func (Raw) ListIterator() datamodel.ListIterator {
	return nil
}
func (Raw) Length() int64 {
	return -1
}
func (Raw) IsAbsent() bool {
	return false
}
func (Raw) IsNull() bool {
	return false
}
func (Raw) AsBool() (bool, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.AsBool()
}
func (Raw) AsInt() (int64, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.AsInt()
}
func (Raw) AsFloat() (float64, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.AsFloat()
}
func (n Raw) AsString() (string, error) {
	return encodeBase64Url(n.x), nil
}
func (n Raw) AsBytes() ([]byte, error) {
	return n.x, nil
}
func (Raw) AsLink() (datamodel.Link, error) {
	return mixins.Bytes{TypeName: "dagjose.Raw"}.AsLink()
}
func (Raw) Prototype() datamodel.NodePrototype {
	return _Raw__Prototype{}
}

type _Raw__Prototype struct{}

func (_Raw__Prototype) NewBuilder() datamodel.NodeBuilder {
	var nb _Raw__Builder
	nb.Reset()
	return &nb
}

type _Raw__Builder struct {
	_Raw__Assembler
}

func (nb *_Raw__Builder) Build() datamodel.Node {
	if *nb.m != schema.Maybe_Value {
		panic("invalid state: cannot call Build on an assembler that's not finished")
	}
	return nb.w
}
func (nb *_Raw__Builder) Reset() {
	var w _Raw
	var m schema.Maybe
	*nb = _Raw__Builder{_Raw__Assembler{w: &w, m: &m}}
}

type _Raw__Assembler struct {
	w *_Raw
	m *schema.Maybe
}

func (na *_Raw__Assembler) reset() {}
func (_Raw__Assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.BeginMap(0)
}
func (_Raw__Assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.BeginList(0)
}
func (na *_Raw__Assembler) AssignNull() error {
	switch *na.m {
	case allowNull:
		*na.m = schema.Maybe_Null
		return nil
	case schema.Maybe_Absent:
		return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.AssignNull()
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	panic("unreachable")
}
func (_Raw__Assembler) AssignBool(bool) error {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.AssignBool(false)
}
func (_Raw__Assembler) AssignInt(int64) error {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.AssignInt(0)
}
func (_Raw__Assembler) AssignFloat(float64) error {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.AssignFloat(0)
}
func (na *_Raw__Assembler) AssignString(v string) error {
	switch *na.m {
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	if decodedBytes, err := decodeBase64Url(v); err != nil {
		return err
	} else {
		na.w.x = decodedBytes
		*na.m = schema.Maybe_Value
		return nil
	}
}
func (na *_Raw__Assembler) AssignBytes(v []byte) error {
	switch *na.m {
	case schema.Maybe_Value, schema.Maybe_Null:
		panic("invalid state: cannot assign into assembler that's already finished")
	}
	na.w.x = v
	*na.m = schema.Maybe_Value
	return nil
}
func (_Raw__Assembler) AssignLink(datamodel.Link) error {
	return mixins.BytesAssembler{TypeName: "dagjose.Raw"}.AssignLink(nil)
}
func (na *_Raw__Assembler) AssignNode(v datamodel.Node) error {
	if v.IsNull() {
		return na.AssignNull()
	}
	if v2, ok := v.(*_Raw); ok {
		switch *na.m {
		case schema.Maybe_Value, schema.Maybe_Null:
			panic("invalid state: cannot assign into assembler that's already finished")
		}
		*na.w = *v2
		*na.m = schema.Maybe_Value
		return nil
	}
	if v2, err := v.AsBytes(); err != nil {
		if e, wrongKind := err.(datamodel.ErrWrongKind); wrongKind && (e.ActualKind == datamodel.Kind_String) {
			if v2, err := v.AsString(); err != nil {
				return err
			} else {
				return na.AssignString(v2)
			}
		}
		return err
	} else {
		return na.AssignBytes(v2)
	}
}
func (_Raw__Assembler) Prototype() datamodel.NodePrototype {
	return _Raw__Prototype{}
}
func (Raw) Type() schema.Type {
	return nil
}
func (n Raw) Representation() datamodel.Node {
	return (*_Raw__Repr)(n)
}

type _Raw__Repr = _Raw

var _ datamodel.Node = &_Raw__Repr{}

type _Raw__ReprPrototype = _Raw__Prototype
type _Raw__ReprAssembler = _Raw__Assembler

func encodeBase64Url(decoded []byte) string {
	return base64.RawURLEncoding.EncodeToString(decoded)
}

func decodeBase64Url(encoded string) ([]byte, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(encoded)
	return decodedBytes, err
}
