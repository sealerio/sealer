package schemadmt

import (
	"fmt"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

// Compile transforms a description of a schema in raw data model ("dmt") form
// into a compiled schema.TypeSystem, which is the ready-to-use form.
//
// The first parameter is mutated by this process,
// and the second parameter is the data source.
//
// The compilation process includes first inserting the "prelude" types into the
// schema.TypeSystem -- that is, the "type Bool bool" and "type String string", etc,
// which are generally presumed to be present in any type system.
//
// The compilation process attempts to check the validity of the schema at a logical level as it goes.
// For example, references to type names not present elsewhere in the same schema are now an error
// (even though that has been easily representable in the dmt.Schema form up until this point).
//
// Note that this API is EXPERIMENTAL and will likely change.
// It supports many features of IPLD Schemas,
// but it may yet not support all of them.
// It supports several validations for logical coherency of schemas,
// but may not yet successfully reject all invalid schemas.
func Compile(ts *schema.TypeSystem, node *Schema) error {
	// Prelude; probably belongs elsewhere.
	{
		ts.Accumulate(schema.SpawnBool("Bool"))
		ts.Accumulate(schema.SpawnInt("Int"))
		ts.Accumulate(schema.SpawnFloat("Float"))
		ts.Accumulate(schema.SpawnString("String"))
		ts.Accumulate(schema.SpawnBytes("Bytes"))

		ts.Accumulate(schema.SpawnAny("Any"))

		ts.Accumulate(schema.SpawnMap("Map", "String", "Any", false))
		ts.Accumulate(schema.SpawnList("List", "Any", false))

		// Should be &Any, really.
		ts.Accumulate(schema.SpawnLink("Link"))

		// TODO: schema package lacks support?
		// ts.Accumulate(schema.SpawnUnit("Null", NullRepr))
	}

	for _, name := range node.Types.Keys {
		defn := node.Types.Values[name]

		// TODO: once ./schema supports anonymous/inline types, remove the ts argument.
		typ, err := spawnType(ts, name, defn)
		if err != nil {
			return err
		}
		ts.Accumulate(typ)
	}

	// TODO: if this fails and the user forgot to check Compile's returned error,
	// we can leave the TypeSystem in an unfortunate broken state:
	// they can obtain types out of the TypeSystem and they are non-nil,
	// but trying to use them in any way may result in panics.
	// Consider making that less prone to misuse, such as making it illegal to
	// call TypeByName until ValidateGraph is happy.
	if errs := ts.ValidateGraph(); errs != nil {
		// Return the first error.
		for _, err := range errs {
			return err
		}
	}
	return nil
}

// Note that the parser and compiler support defaults. We're lacking support in bindnode.
func todoFromImplicitlyFalseBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func anonTypeName(nameOrDefn TypeNameOrInlineDefn) string {
	if nameOrDefn.TypeName != nil {
		return *nameOrDefn.TypeName
	}
	defn := *nameOrDefn.InlineDefn
	switch {
	case defn.TypeDefnMap != nil:
		defn := defn.TypeDefnMap
		return fmt.Sprintf("Map__%s__%s", defn.KeyType, anonTypeName(defn.ValueType))
	case defn.TypeDefnList != nil:
		defn := defn.TypeDefnList
		return fmt.Sprintf("List__%s", anonTypeName(defn.ValueType))
	case defn.TypeDefnLink != nil:
		return anonLinkName(*defn.TypeDefnLink)
	default:
		panic(fmt.Errorf("%#v", defn))
	}
}

func anonLinkName(defn TypeDefnLink) string {
	if defn.ExpectedType != nil {
		return fmt.Sprintf("Link__%s", *defn.ExpectedType)
	}
	return "Link__Link"
}

func parseKind(s string) datamodel.Kind {
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
		return datamodel.Kind_Invalid
	}
}

func spawnType(ts *schema.TypeSystem, name schema.TypeName, defn TypeDefn) (schema.Type, error) {
	switch {
	// Scalar types without parameters.
	case defn.TypeDefnBool != nil:
		return schema.SpawnBool(name), nil
	case defn.TypeDefnString != nil:
		return schema.SpawnString(name), nil
	case defn.TypeDefnBytes != nil:
		return schema.SpawnBytes(name), nil
	case defn.TypeDefnInt != nil:
		return schema.SpawnInt(name), nil
	case defn.TypeDefnFloat != nil:
		return schema.SpawnFloat(name), nil

	case defn.TypeDefnList != nil:
		typ := defn.TypeDefnList
		tname := ""
		if typ.ValueType.TypeName != nil {
			tname = *typ.ValueType.TypeName
		} else if tname = anonTypeName(typ.ValueType); ts.TypeByName(tname) == nil {
			anonDefn := TypeDefn{
				TypeDefnMap:  typ.ValueType.InlineDefn.TypeDefnMap,
				TypeDefnList: typ.ValueType.InlineDefn.TypeDefnList,
				TypeDefnLink: typ.ValueType.InlineDefn.TypeDefnLink,
			}
			anonType, err := spawnType(ts, tname, anonDefn)
			if err != nil {
				return nil, err
			}
			ts.Accumulate(anonType)
		}
		switch {
		case typ.Representation == nil ||
			typ.Representation.ListRepresentation_List != nil:
			// default behavior
		default:
			return nil, fmt.Errorf("TODO: support other list repr in schema package")
		}
		return schema.SpawnList(name,
			tname,
			todoFromImplicitlyFalseBool(typ.ValueNullable),
		), nil
	case defn.TypeDefnMap != nil:
		typ := defn.TypeDefnMap
		tname := ""
		if typ.ValueType.TypeName != nil {
			tname = *typ.ValueType.TypeName
		} else if tname = anonTypeName(typ.ValueType); ts.TypeByName(tname) == nil {
			anonDefn := TypeDefn{
				TypeDefnMap:  typ.ValueType.InlineDefn.TypeDefnMap,
				TypeDefnList: typ.ValueType.InlineDefn.TypeDefnList,
				TypeDefnLink: typ.ValueType.InlineDefn.TypeDefnLink,
			}
			anonType, err := spawnType(ts, tname, anonDefn)
			if err != nil {
				return nil, err
			}
			ts.Accumulate(anonType)
		}
		switch {
		case typ.Representation == nil ||
			typ.Representation.MapRepresentation_Map != nil:
			// default behavior
		case typ.Representation.MapRepresentation_Stringpairs != nil:
			return nil, fmt.Errorf("TODO: support stringpairs map repr in schema package")
		default:
			return nil, fmt.Errorf("TODO: support other map repr in schema package")
		}
		return schema.SpawnMap(name,
			typ.KeyType,
			tname,
			todoFromImplicitlyFalseBool(typ.ValueNullable),
		), nil
	case defn.TypeDefnStruct != nil:
		typ := defn.TypeDefnStruct
		var fields []schema.StructField
		for _, fname := range typ.Fields.Keys {
			field := typ.Fields.Values[fname]
			tname := ""
			if field.Type.TypeName != nil {
				tname = *field.Type.TypeName
			} else if tname = anonTypeName(field.Type); ts.TypeByName(tname) == nil {
				// Note that TypeDefn and InlineDefn aren't the same enum.
				anonDefn := TypeDefn{
					TypeDefnMap:  field.Type.InlineDefn.TypeDefnMap,
					TypeDefnList: field.Type.InlineDefn.TypeDefnList,
					TypeDefnLink: field.Type.InlineDefn.TypeDefnLink,
				}
				anonType, err := spawnType(ts, tname, anonDefn)
				if err != nil {
					return nil, err
				}
				ts.Accumulate(anonType)
			}
			fields = append(fields, schema.SpawnStructField(fname,
				tname,
				todoFromImplicitlyFalseBool(field.Optional),
				todoFromImplicitlyFalseBool(field.Nullable),
			))
		}
		var repr schema.StructRepresentation
		switch {
		case typ.Representation.StructRepresentation_Map != nil:
			rp := typ.Representation.StructRepresentation_Map
			if rp.Fields == nil {
				repr = schema.SpawnStructRepresentationMap2(nil, nil)
				break
			}
			renames := make(map[string]string, len(rp.Fields.Keys))
			implicits := make(map[string]schema.ImplicitValue, len(rp.Fields.Keys))
			for _, name := range rp.Fields.Keys {
				details := rp.Fields.Values[name]
				if details.Rename != nil {
					renames[name] = *details.Rename
				}
				if imp := details.Implicit; imp != nil {
					var sumVal schema.ImplicitValue
					switch {
					case imp.Bool != nil:
						sumVal = schema.ImplicitValue_Bool(*imp.Bool)
					case imp.String != nil:
						sumVal = schema.ImplicitValue_String(*imp.String)
					case imp.Int != nil:
						sumVal = schema.ImplicitValue_Int(*imp.Int)
					default:
						panic("TODO: implicit value kind")
					}
					implicits[name] = sumVal
				}

			}
			repr = schema.SpawnStructRepresentationMap2(renames, implicits)
		case typ.Representation.StructRepresentation_Tuple != nil:
			rp := typ.Representation.StructRepresentation_Tuple
			if rp.FieldOrder == nil {
				repr = schema.SpawnStructRepresentationTuple()
				break
			}
			return nil, fmt.Errorf("TODO: support for tuples with field orders in the schema package")
		case typ.Representation.StructRepresentation_Stringjoin != nil:
			join := typ.Representation.StructRepresentation_Stringjoin.Join
			if join == "" {
				return nil, fmt.Errorf("stringjoin has empty join value")
			}
			repr = schema.SpawnStructRepresentationStringjoin(join)
		case typ.Representation.StructRepresentation_Listpairs != nil:
			repr = schema.SpawnStructRepresentationListPairs()
		default:
			return nil, fmt.Errorf("TODO: support other struct repr in schema package")
		}
		return schema.SpawnStruct(name,
			fields,
			repr,
		), nil
	case defn.TypeDefnUnion != nil:
		typ := defn.TypeDefnUnion
		var members []schema.TypeName
		for _, member := range typ.Members {
			if member.TypeName != nil {
				members = append(members, *member.TypeName)
			} else {
				tname := anonLinkName(*member.UnionMemberInlineDefn.TypeDefnLink)
				members = append(members, tname)
				if ts.TypeByName(tname) == nil {
					anonDefn := TypeDefn{
						TypeDefnLink: member.UnionMemberInlineDefn.TypeDefnLink,
					}
					anonType, err := spawnType(ts, tname, anonDefn)
					if err != nil {
						return nil, err
					}
					ts.Accumulate(anonType)
				}
			}
		}
		remainingMembers := make(map[string]bool)
		for _, memberName := range members {
			remainingMembers[memberName] = true
		}
		validMember := func(memberName string) error {
			switch remaining, known := remainingMembers[memberName]; {
			case remaining:
				remainingMembers[memberName] = false
				return nil
			case !known:
				return fmt.Errorf("%q is not a valid member of union %q", memberName, name)
			default:
				return fmt.Errorf("%q is duplicate in the union repr of %q", memberName, name)
			}
		}

		var repr schema.UnionRepresentation
		switch {
		case typ.Representation.UnionRepresentation_Kinded != nil:
			rp := typ.Representation.UnionRepresentation_Kinded
			table := make(map[datamodel.Kind]schema.TypeName, len(rp.Keys))
			for _, kindStr := range rp.Keys {
				kind := parseKind(kindStr)
				member := rp.Values[kindStr]
				switch {
				case member.TypeName != nil:
					memberName := *member.TypeName
					if err := validMember(memberName); err != nil {
						return nil, err
					}
					table[kind] = memberName
				case member.UnionMemberInlineDefn != nil:
					tname := anonLinkName(*member.UnionMemberInlineDefn.TypeDefnLink)
					if err := validMember(tname); err != nil {
						return nil, err
					}
					table[kind] = tname
				}
			}
			repr = schema.SpawnUnionRepresentationKinded(table)
		case typ.Representation.UnionRepresentation_Keyed != nil:
			rp := typ.Representation.UnionRepresentation_Keyed
			table := make(map[string]schema.TypeName, len(rp.Keys))
			for _, key := range rp.Keys {
				member := rp.Values[key]
				switch {
				case member.TypeName != nil:
					memberName := *member.TypeName
					if err := validMember(memberName); err != nil {
						return nil, err
					}
					table[key] = memberName
				case member.UnionMemberInlineDefn != nil:
					tname := anonLinkName(*member.UnionMemberInlineDefn.TypeDefnLink)
					if err := validMember(tname); err != nil {
						return nil, err
					}
					table[key] = tname
				}
			}
			repr = schema.SpawnUnionRepresentationKeyed(table)
		case typ.Representation.UnionRepresentation_StringPrefix != nil:
			prefixes := typ.Representation.UnionRepresentation_StringPrefix.Prefixes
			for _, key := range prefixes.Keys {
				if err := validMember(prefixes.Values[key]); err != nil {
					return nil, err
				}
			}
			repr = schema.SpawnUnionRepresentationStringprefix("", prefixes.Values)
		case typ.Representation.UnionRepresentation_Inline != nil:
			rp := typ.Representation.UnionRepresentation_Inline
			if rp.DiscriminantKey == "" {
				return nil, fmt.Errorf("inline union has empty discriminantKey value")
			}
			if rp.DiscriminantTable.Keys == nil || rp.DiscriminantTable.Values == nil {
				return nil, fmt.Errorf("inline union has empty discriminantTable")
			}
			for _, key := range rp.DiscriminantTable.Keys {
				if err := validMember(rp.DiscriminantTable.Values[key]); err != nil {
					return nil, err
				}
			}
			repr = schema.SpawnUnionRepresentationInline(rp.DiscriminantKey, rp.DiscriminantTable.Values)
		default:
			return nil, fmt.Errorf("TODO: support other union repr in schema package")
		}
		for memberName, remaining := range remainingMembers {
			if remaining {
				return nil, fmt.Errorf("%q is not present in the union repr of %q", memberName, name)
			}
		}
		return schema.SpawnUnion(name,
			members,
			repr,
		), nil
	case defn.TypeDefnEnum != nil:
		typ := defn.TypeDefnEnum
		var repr schema.EnumRepresentation

		// TODO: we should probably also reject duplicates.
		validMember := func(name string) bool {
			for _, memberName := range typ.Members {
				if memberName == name {
					return true
				}
			}
			return false
		}
		switch {
		case typ.Representation.EnumRepresentation_String != nil:
			rp := typ.Representation.EnumRepresentation_String
			for memberName := range rp.Values {
				if !validMember(memberName) {
					return nil, fmt.Errorf("%q is not a valid member of enum %q", memberName, name)
				}
			}
			repr = schema.EnumRepresentation_String(rp.Values)
		case typ.Representation.EnumRepresentation_Int != nil:
			rp := typ.Representation.EnumRepresentation_Int
			for memberName := range rp.Values {
				if !validMember(memberName) {
					return nil, fmt.Errorf("%q is not a valid member of enum %q", memberName, name)
				}
			}
			repr = schema.EnumRepresentation_Int(rp.Values)
		default:
			return nil, fmt.Errorf("TODO: support other enum repr in schema package")
		}
		return schema.SpawnEnum(name,
			typ.Members,
			repr,
		), nil
	case defn.TypeDefnLink != nil:
		typ := defn.TypeDefnLink
		if typ.ExpectedType == nil {
			return schema.SpawnLink(name), nil
		}
		return schema.SpawnLinkReference(name, *typ.ExpectedType), nil
	case defn.TypeDefnAny != nil:
		return schema.SpawnAny(name), nil
	default:
		panic(fmt.Errorf("%#v", defn))
	}
}
