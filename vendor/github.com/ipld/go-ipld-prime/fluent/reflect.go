package fluent

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/ipld/go-ipld-prime/datamodel"
)

// Reflect creates a new Node by looking at a golang value with reflection
// and converting it into IPLD Data Model.
// This is a quick-and-dirty way to get data into the IPLD Data Model;
// it's useful for rapid prototyping and demos,
// but note that this feature is not intended to be suitable for "production" use
// due to low performance and lack of configurability.
//
// The concrete type of the returned Node is determined by
// the NodePrototype argument provided by the caller.
//
// No type information from the golang value will be observable in the result.
//
// The reflection will walk over any golang value, but is not configurable.
// Golang maps become IPLD maps; golang slices and arrays become IPLD lists;
// and golang structs become IPLD maps too.
// When converting golang structs to IPLD maps, the field names will become the map keys.
// Pointers and interfaces will be traversed transparently and are not visible in the output.
//
// An error will be returned if the process of assembling the Node returns any errors
// (for example, if the NodePrototype is for a schema-constrained Node,
// any validation errors from the schema will cause errors to be returned).
//
// A panic will be raised if there is any difficulty examining the golang value via reflection
// (for example, if the value is a struct with unexported fields,
// or if a non-data type like a channel or function is encountered).
//
// Some configuration (in particular, what to do about map ordering) is available via the Reflector struct.
// That structure has a method of the same name and signiture as this one on it.
// (This function is a shortcut for calling that method on a Reflector struct with default configuration.)
//
// Performance remarks: performance of this function will generally be poor.
// In general, creating data in golang types and then *flipping* it to IPLD form
// involves handling the data at least twice, and so will always be slower
// than just creating the same data in IPLD form programmatically directly.
// In particular, reflection is generally not fast, and this feature has
// not been optimized for either speed nor allocation avoidance.
// Other features in the fluent package will typically out-perform this,
// and using NodeAssemblers directly (without any fluent tools) will be much faster.
// Only use this function if performance is not of consequence.
func Reflect(np datamodel.NodePrototype, i interface{}) (datamodel.Node, error) {
	return defaultReflector.Reflect(np, i)
}

// MustReflect is a shortcut for Reflect but panics on any error.
// It is useful if you need a single return value for function composition purposes.
func MustReflect(np datamodel.NodePrototype, i interface{}) datamodel.Node {
	n, err := Reflect(np, i)
	if err != nil {
		panic(err)
	}
	return n
}

// ReflectIntoAssembler is similar to Reflect, but takes a NodeAssembler parameter
// instead of a Node Prototype.
// This may be useful if you need more direct control over allocations,
// or want to fill in only part of a larger node assembly process using the reflect tool.
// Data is accumulated by the NodeAssembler parameter, so no Node is returned.
func ReflectIntoAssembler(na datamodel.NodeAssembler, i interface{}) error {
	return defaultReflector.ReflectIntoAssembler(na, i)
}

var defaultReflector = Reflector{
	MapOrder: func(x, y string) bool {
		return x < y
	},
}

// Reflector allows configuration of the Reflect family of functions
// (`Reflect`, `ReflectIntoAssembler`, etc).
type Reflector struct {
	// MapOrder is used to decide a deterministic order for inserting entries to maps.
	// (This is used when converting golang maps, since their iteration order is randomized;
	// it is not used when converting other types such as structs, since those have a stable order.)
	// MapOrder should return x < y in the same way as sort.Interface.Less.
	//
	// If using a default Reflector (e.g. via the package-scope functions),
	// this function is a simple natural golang string sort: it performs `x < y` on the strings.
	MapOrder func(x, y string) bool
}

// Reflect is as per the package-scope function of the same name and signature,
// but using the configuration in the Reflector struct.
// See the package-scope function for documentation.
func (rcfg Reflector) Reflect(np datamodel.NodePrototype, i interface{}) (datamodel.Node, error) {
	nb := np.NewBuilder()
	if err := rcfg.ReflectIntoAssembler(nb, i); err != nil {
		return nil, err
	}
	return nb.Build(), nil
}

// ReflectIntoAssembler is as per the package-scope function of the same name and signature,
// but using the configuration in the Reflector struct.
// See the package-scope function for documentation.
func (rcfg Reflector) ReflectIntoAssembler(na datamodel.NodeAssembler, i interface{}) error {
	// Cover the most common values with a type-switch, as it's faster than reflection.
	switch x := i.(type) {
	case map[string]string:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Sort(sortableStrings{keys, rcfg.MapOrder})
		ma, err := na.BeginMap(int64(len(x)))
		if err != nil {
			return err
		}
		for _, k := range keys {
			va, err := ma.AssembleEntry(k)
			if err != nil {
				return err
			}
			if err := va.AssignString(x[k]); err != nil {
				return err
			}
		}
		return ma.Finish()
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Sort(sortableStrings{keys, rcfg.MapOrder})
		ma, err := na.BeginMap(int64(len(x)))
		if err != nil {
			return err
		}
		for _, k := range keys {
			va, err := ma.AssembleEntry(k)
			if err != nil {
				return err
			}
			if err := rcfg.ReflectIntoAssembler(va, x[k]); err != nil {
				return err
			}
		}
		return ma.Finish()
	case []string:
		la, err := na.BeginList(int64(len(x)))
		if err != nil {
			return err
		}
		for _, v := range x {
			if err := la.AssembleValue().AssignString(v); err != nil {
				return err
			}
		}
		return la.Finish()
	case []interface{}:
		la, err := na.BeginList(int64(len(x)))
		if err != nil {
			return err
		}
		for _, v := range x {
			if err := rcfg.ReflectIntoAssembler(la.AssembleValue(), v); err != nil {
				return err
			}
		}
		return la.Finish()
	case string:
		return na.AssignString(x)
	case []byte:
		return na.AssignBytes(x)
	case int64:
		return na.AssignInt(x)
	case nil:
		return na.AssignNull()
	}
	// That didn't fly?  Reflection time.
	rv := reflect.ValueOf(i)
	switch rv.Kind() {
	case reflect.Bool:
		return na.AssignBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return na.AssignInt(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return na.AssignInt(int64(rv.Uint())) // TODO: check overflow
	case reflect.Float32, reflect.Float64:
		return na.AssignFloat(rv.Float())
	case reflect.String:
		return na.AssignString(rv.String())
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 { // byte slices are a special case
			return na.AssignBytes(rv.Bytes())
		}
		l := rv.Len()
		la, err := na.BeginList(int64(l))
		if err != nil {
			return err
		}
		for i := 0; i < l; i++ {
			if err := rcfg.ReflectIntoAssembler(la.AssembleValue(), rv.Index(i).Interface()); err != nil {
				return err
			}
		}
		return la.Finish()
	case reflect.Map:
		// the keys slice for sorting keeps things in reflect.Value form, because unboxing is cheap,
		//  but re-boxing is not cheap, and the MapIndex method requires reflect.Value again later.
		keys := make([]reflect.Value, 0, rv.Len())
		itr := rv.MapRange()
		for itr.Next() {
			k := itr.Key()
			if k.Kind() != reflect.String {
				return fmt.Errorf("cannot convert a map with non-string keys (%T)", i)
			}
			keys = append(keys, k)
		}
		sort.Sort(sortableReflectStrings{keys, rcfg.MapOrder})
		ma, err := na.BeginMap(int64(rv.Len()))
		if err != nil {
			return err
		}
		for _, k := range keys {
			va, err := ma.AssembleEntry(k.String())
			if err != nil {
				return err
			}
			if err := rcfg.ReflectIntoAssembler(va, rv.MapIndex(k).Interface()); err != nil {
				return err
			}
		}
		return ma.Finish()
	case reflect.Struct:
		l := rv.NumField()
		ma, err := na.BeginMap(int64(l))
		if err != nil {
			return err
		}
		for i := 0; i < l; i++ {
			fn := rv.Type().Field(i).Name
			fv := rv.Field(i)
			va, err := ma.AssembleEntry(fn)
			if err != nil {
				return err
			}
			if err := rcfg.ReflectIntoAssembler(va, fv.Interface()); err != nil {
				return err
			}
		}
		return ma.Finish()
	case reflect.Ptr:
		if rv.IsNil() {
			return na.AssignNull()
		}
		return rcfg.ReflectIntoAssembler(na, rv.Elem())
	case reflect.Interface:
		return rcfg.ReflectIntoAssembler(na, rv.Elem())
	}
	// Some kints of values -- like Uintptr, Complex64/128, Channels, etc -- are not supported by this function.
	return fmt.Errorf("fluent.Reflect: unsure how to handle type %T (kind: %v)", i, rv.Kind())
}

type sortableStrings struct {
	a    []string
	less func(x, y string) bool
}

func (a sortableStrings) Len() int           { return len(a.a) }
func (a sortableStrings) Swap(i, j int)      { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a sortableStrings) Less(i, j int) bool { return a.less(a.a[i], a.a[j]) }

type sortableReflectStrings struct {
	a    []reflect.Value
	less func(x, y string) bool
}

func (a sortableReflectStrings) Len() int           { return len(a.a) }
func (a sortableReflectStrings) Swap(i, j int)      { a.a[i], a.a[j] = a.a[j], a.a[i] }
func (a sortableReflectStrings) Less(i, j int) bool { return a.less(a.a[i].String(), a.a[j].String()) }
