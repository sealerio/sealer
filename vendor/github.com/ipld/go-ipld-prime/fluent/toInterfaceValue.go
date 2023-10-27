package fluent

import (
	"errors"

	"github.com/ipld/go-ipld-prime/datamodel"
)

var errInvalidKind = errors.New("invalid kind")
var errUnknownKind = errors.New("unknown kind")

// ToInterface converts an IPLD node to its simplest equivalent Go value.
//
// Booleans, integers, floats, strings, bytes, and links are returned as themselves,
// as per the node's AsT method. Note that nulls are returned as untyped nils.
//
// Lists and maps are returned as []interface{} and map[string]interface{}, respectively.
func ToInterface(node datamodel.Node) (interface{}, error) {
	switch k := node.Kind(); k {
	case datamodel.Kind_Invalid:
		return nil, errInvalidKind
	case datamodel.Kind_Null:
		return nil, nil
	case datamodel.Kind_Bool:
		return node.AsBool()
	case datamodel.Kind_Int:
		return node.AsInt()
	case datamodel.Kind_Float:
		return node.AsFloat()
	case datamodel.Kind_String:
		return node.AsString()
	case datamodel.Kind_Bytes:
		return node.AsBytes()
	case datamodel.Kind_Link:
		return node.AsLink()
	case datamodel.Kind_Map:
		outMap := make(map[string]interface{}, node.Length())
		for mi := node.MapIterator(); !mi.Done(); {
			k, v, err := mi.Next()
			if err != nil {
				return nil, err
			}
			kVal, err := k.AsString()
			if err != nil {
				return nil, err
			}
			vVal, err := ToInterface(v)
			if err != nil {
				return nil, err
			}
			outMap[kVal] = vVal
		}
		return outMap, nil
	case datamodel.Kind_List:
		outList := make([]interface{}, 0, node.Length())
		for li := node.ListIterator(); !li.Done(); {
			_, v, err := li.Next()
			if err != nil {
				return nil, err
			}
			vVal, err := ToInterface(v)
			if err != nil {
				return nil, err
			}
			outList = append(outList, vVal)
		}
		return outList, nil
	default:
		return nil, errUnknownKind
	}
}
