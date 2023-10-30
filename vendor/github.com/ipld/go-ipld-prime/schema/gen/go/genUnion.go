package gengo

import (
	"io"

	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/schema/gen/go/mixins"
)

// The generator for unions is a bit more wild than most others:
// it has at three major branches for how its internals are laid out:
//
//   - all possible children are embedded.
//   - all possible children are pointers... in which case we collapse to one interface resident.
//       (n.b. this does give up some inlining potential as well as gives up on alloc amortization, but it does make resident memory size minimal.)
//   - some children are emebedded and some are pointers, and of the latter set, they may be either in one interface field or several discrete pointers.
//       (discrete fields of pointer type makes inlining possible in some paths, whereas an interface field blocks it).
//
// ... We're not doing that last one at all right now.  The pareto-prevalence of these concerns is extremely low compared to the effort required.
// But the first two are both very reasonable, and both are often wanted.
//
// These choices are made from adjunct config (which should make sense, because they're clearly all "golang" details -- not type semantics).
// We still tackle all the generation for all these strategies this in one file,
//  because all of the interfaces we export are the same, regardless of the internals (and it just seems easiest to do this way).

type unionGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

// --- native content and specializations --->

func (g unionGenerator) EmitNativeType(w io.Writer) {
	// We generate *two* types: a struct which acts as the union node,
	// and also an interface which covers the members (and has an unexported marker function to make sure the set can't be extended).
	//
	// The interface *mostly* isn't used... except for in the return type of a speciated function which can be used to do golang-native type switches.
	//
	// The interface also includes a requirement for an errorless primitive access method (such as `String() string`)
	// if our representation strategy is one that has that semantic (e.g., stringprefix repr does).
	//
	// A note about index: in all cases the index of a member type is used, we increment it by one, to avoid using zero.
	// We do this because it's desirable to reserve the zero in the 'tag' field (if we generate one) as a sentinel value
	// (see further comments in the EmitNodeAssemblerType function);
	// and since we do it in that one case, it's just as well to do it uniformly.
	doTemplate(`
		{{- if Comments -}}
		// {{ .Type | TypeSymbol }} matches the IPLD Schema type "{{ .Type.Name }}".
		// {{ .Type | TypeSymbol }} has {{ .Type.TypeKind }} typekind, which means its data model behaviors are that of a {{ .Kind }} kind.
		{{- end}}
		type {{ .Type | TypeSymbol }} = *_{{ .Type | TypeSymbol }}
		type _{{ .Type | TypeSymbol }} struct {
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			tag uint
			{{- range $i, $member := .Type.Members }}
			x{{ add $i 1 }} _{{ $member | TypeSymbol }}
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			x _{{ .Type | TypeSymbol }}__iface
			{{- end}}
		}
		type _{{ .Type | TypeSymbol }}__iface interface {
			_{{ .Type | TypeSymbol }}__member()
			{{- if (eq (.Type.RepresentationStrategy | printf "%T") "schema.UnionRepresentation_Stringprefix") }}
			String() string
			{{- end}}
		}

		{{- range $member := .Type.Members }}
		func (_{{ $member | TypeSymbol }}) _{{ dot.Type | TypeSymbol }}__member() {}
		{{- end}}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNativeAccessors(w io.Writer) {
	doTemplate(`
		func (n _{{ .Type | TypeSymbol }}) AsInterface() _{{ .Type | TypeSymbol }}__iface {
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch n.tag {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				return &n.x{{ add $i 1 }}
			{{- end}}
			default:
				panic("invalid union state; how did you create this object?")
			}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			return n.x
			{{- end}}
		}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNativeBuilder(w io.Writer) {
	// Unclear as yet what should go here.
}

func (g unionGenerator) EmitNativeMaybe(w io.Writer) {
	emitNativeMaybe(w, g.AdjCfg, g)
}

// --- type info --->

func (g unionGenerator) EmitTypeConst(w io.Writer) {
	doTemplate(`
		// TODO EmitTypeConst
	`, w, g.AdjCfg, g)
}

// --- TypedNode interface satisfaction --->

func (g unionGenerator) EmitTypedNodeMethodType(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Type() schema.Type {
			return nil /*TODO:typelit*/
		}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitTypedNodeMethodRepresentation(w io.Writer) {
	emitTypicalTypedNodeMethodRepresentation(w, g.AdjCfg, g)
}

// --- Node interface satisfaction --->

func (g unionGenerator) EmitNodeType(w io.Writer) {
	// No additional types needed.  Methods all attach to the native type.

	// We do, however, want some constants for our member names;
	//  they'll make iterators able to work faster.  So let's emit those.
	// These are a bit perplexing, because they're... type names.
	//  However, oddly enough, we don't have type names available *as nodes* anywhere else centrally available,
	//   so... we generate some values for them here with scoped identifers and get on with it.
	//    Maybe this could be elided with future work.
	doTemplate(`
		var (
			{{- range $member := .Type.Members }}
			memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }} = _String{"{{ $member.Name }}"}
			{{- end }}
		)
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeTypeAssertions(w io.Writer) {
	emitNodeTypeAssertions_typical(w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeMethodLookupByString(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByString(key string) (datamodel.Node, error) {
			switch key {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member.Name }}":
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				if n.tag != {{ add $i 1 }} {
					return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
				}
				return &n.x{{ add $i 1 }}, nil
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				if n2, ok := n.x.({{ $member | TypeSymbol }}); ok {
					return n2, nil
				} else {
					return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
				}
				{{- end}}
			{{- end}}
			default:
				return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: datamodel.PathSegmentOfString(key)}
			}
		}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeMethodLookupByNode(w io.Writer) {
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
			ks, err := key.AsString()
			if err != nil {
				return nil, err
			}
			return n.LookupByString(ks)
		}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeMethodMapIterator(w io.Writer) {
	// This is kind of a hilarious "iterator": it has to count all the way up to... 1.
	doTemplate(`
		func (n {{ .Type | TypeSymbol }}) MapIterator() datamodel.MapIterator {
			return &_{{ .Type | TypeSymbol }}__MapItr{n, false}
		}

		type _{{ .Type | TypeSymbol }}__MapItr struct {
			n {{ .Type | TypeSymbol }}
			done bool
		}

		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Next() (k datamodel.Node, v datamodel.Node, _ error) {
			if itr.done {
				return nil, nil, datamodel.ErrIteratorOverread{}
			}
			{{- if (eq (.AdjCfg.UnionMemlayout .Type) "embedAll") }}
			switch itr.n.tag {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				k, v = &memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}, &itr.n.x{{ add $i 1 }}
			{{- end}}
			{{- else if (eq (.AdjCfg.UnionMemlayout .Type) "interface") }}
			switch n2 := itr.n.x.(type) {
			{{- range $member := .Type.Members }}
			case {{ $member | TypeSymbol }}:
				k, v = &memberName__{{ dot.Type | TypeSymbol }}_{{ $member.Name }}, n2
			{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
			itr.done = true
			return
		}
		func (itr *_{{ .Type | TypeSymbol }}__MapItr) Done() bool {
			return itr.done
		}

	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeMethodLength(w io.Writer) {
	doTemplate(`
		func ({{ .Type | TypeSymbol }}) Length() int64 {
			return 1
		}
	`, w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodeMethodPrototype(w io.Writer) {
	emitNodeMethodPrototype_typical(w, g.AdjCfg, g)
}

func (g unionGenerator) EmitNodePrototypeType(w io.Writer) {
	emitNodePrototypeType_typical(w, g.AdjCfg, g)
}

// --- NodeBuilder and NodeAssembler --->

func (g unionGenerator) GetNodeBuilderGenerator() NodeBuilderGenerator {
	return unionBuilderGenerator{
		g.AdjCfg,
		mixins.MapAssemblerTraits{
			PkgName:       g.PkgName,
			TypeName:      g.TypeName,
			AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__",
		},
		g.PkgName,
		g.Type,
	}
}

type unionBuilderGenerator struct {
	AdjCfg *AdjunctCfg
	mixins.MapAssemblerTraits
	PkgName string
	Type    *schema.TypeUnion
}

func (unionBuilderGenerator) IsRepr() bool { return false } // hint used in some generalized templates.

func (g unionBuilderGenerator) EmitNodeBuilderType(w io.Writer) {
	emitEmitNodeBuilderType_typical(w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeBuilderMethods(w io.Writer) {
	emitNodeBuilderMethods_typical(w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeAssemblerType(w io.Writer) {
	// Assemblers for unions are not unlikely those for structs or maps:
	//
	// - 'w' is the "**w**ip" pointer.
	// - 'm' is the pointer to a **m**aybe which communicates our completeness to the parent if we're a child assembler.
	//     Like any other structure, a union can be nullable in the context of some enclosing object, and we'll have the usual branches for handling that in our various Assign methods.
	// - 'state' is what it says on the tin.  Unions use maState to sequence the transitions between a new assembler, the map having been started, key insertions, value insertions, and finish.
	//     Most of this is just like the way struct and map use maState.
	//     However, we also need to guard to make sure a second entry never begins; after the first, finish is the *only* valid transition.
	//     In structs, this is done using the "set" bitfield; in maps, the state resides in the wip map itself.
	//     Unions are more like the latter: depending on which memory layout we're using, either the `na.w.tag` value, or, a non-nil `na.w.x`, is indicative that one key has been entered.
	//     (The zero value for `na.w.tag` is reserved, and all  for this reason.
	// - There is no additional state need to store "focus" (in contrast to structs);
	//     information during the AssembleValue phase about which member is selected is also just handled in `na.w.tag`, or, in the type info of `na.w.x`, again depending on memory layout strategy.
	//     (This is subverted a bit by the 'ca' field, however... which effectively mirrors `na.w.tag`, and is only active in the resetting process, but is necessary because it outlives its twin inside 'w'.)
	//
	// - 'cm' is **c**hild **m**aybe and is used for the completion message from children.
	// - 'ca*' fields embed **c**hild **a**ssemblers -- these are embedded so we can yield pointers to them during recusion into child value assembly without causing new allocations.
	//     In unions, only one of these will every be used!  However, we don't know *which one* in advance, so, we have to embed them all.
	//     (It's ironic to note that if the golang compiler had an understanding of unions itself (either tagged or untagged would suffice), we could compile this down into *much* more minimal amounts of resident memory reservation.  Alas!)
	//     The 'ca*' fields are pointers (and allocated on demand) instead of embeds for unions with memlayout=interface mode.  (Arguably, this is overloading that config; PRs for more granular configurability welcome.)
	// - 'ca' (with no further suffix) identifies which child assembler was previously used.
	//     This is for minimizing the amount of work that resetting has to do: it will only recurse into resetting that child assembler.
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__Assembler struct {
			w *_{{ .Type | TypeSymbol }}
			m *schema.Maybe
			state maState

			cm schema.Maybe
			{{- range $i, $member := .Type.Members }}
			ca{{ add $i 1 }} {{ if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}*{{end}}_{{ $member | TypeSymbol }}__Assembler
			{{end -}}
			ca uint
		}
	`, w, g.AdjCfg, g)

	// Reset methods for unions are a tad more involved than for most other assemblers:
	//  we only want to bother to reset whichever child assembler (if any) we actually used last.
	//  We *could* blithely reset *all* child assemblers every time; but, trading an extra bit of state in our assembler
	//   for the privledge of trimming off a potentially sizable amount of unnecessary zeroing efforts seems preferrable.
	//  Also, although go syntax makes it not textually obvious here, note that it's possible for the child assemblers to be either pointers or embeds:
	//   on consequence of this is that just zeroing this struct would be both unreliable and undesirable in the pointer case
	//    (it would leave orphan child assemblers that might still have pointers into us, which could be guarded against but is nonetheless is considerably scary in complexity;
	//    and it would also mean that we can't keep ahold of the child assemblers across resets and thus amortize allocations, which... is the whole reason the reset system exists in the first place).
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__Assembler) reset() {
			na.state = maState_initial
			switch na.ca {
			case 0:
				return
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				na.ca{{ add $i 1 }}.reset()
			{{end -}}
			default:
				panic("unreachable")
			}
			na.ca = 0
			na.cm = schema.Maybe_Absent
		}
	`, w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeAssemblerMethodBeginMap(w io.Writer) {
	emitNodeAssemblerMethodBeginMap_strictoid(w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeAssemblerMethodAssignNull(w io.Writer) {
	// It might sound a bit odd to call a union "recursive", since it's so very trivially so (no fan-out),
	//  but it's functionally accurate: the generated method should include a branch for the 'midvalue' state.
	emitNodeAssemblerMethodAssignNull_recursive(w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeAssemblerMethodAssignNode(w io.Writer) {
	// AssignNode goes through three phases:
	// 1. is it null?  Jump over to AssignNull (which may or may not reject it).
	// 2. is it our own type?  Handle specially -- we might be able to do efficient things.
	// 3. is it the right kind to morph into us?  Do so.
	//
	// We do not set m=midvalue in phase 3 -- it shouldn't matter unless you're trying to pull off concurrent access, which is wrong and unsafe regardless.
	//
	// DRY: this turns out to be textually identical to the method for structs!  (At least, for now.  It could/should probably be optimized to get to the point faster in phase 3.)
	doTemplate(`
		func (na *_{{ .Type | TypeSymbol }}__Assembler) AssignNode(v datamodel.Node) error {
			if v.IsNull() {
				return na.AssignNull()
			}
			if v2, ok := v.(*_{{ .Type | TypeSymbol }}); ok {
				switch *na.m {
				case schema.Maybe_Value, schema.Maybe_Null:
					panic("invalid state: cannot assign into assembler that's already finished")
				case midvalue:
					panic("invalid state: cannot assign null into an assembler that's already begun working on recursive structures!")
				}
				{{- if .Type | MaybeUsesPtr }}
				if na.w == nil {
					na.w = v2
					*na.m = schema.Maybe_Value
					return nil
				}
				{{- end}}
				*na.w = *v2
				*na.m = schema.Maybe_Value
				return nil
			}
			if v.Kind() != datamodel.Kind_Map {
				return datamodel.ErrWrongKind{TypeName: "{{ .PkgName }}.{{ .Type.Name }}", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustMap, ActualKind: v.Kind()}
			}
			itr := v.MapIterator()
			for !itr.Done() {
				k, v, err := itr.Next()
				if err != nil {
					return err
				}
				if err := na.AssembleKey().AssignNode(k); err != nil {
					return err
				}
				if err := na.AssembleValue().AssignNode(v); err != nil {
					return err
				}
			}
			return na.Finish()
		}
	`, w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) EmitNodeAssemblerOtherBits(w io.Writer) {
	g.emitMapAssemblerChildTidyHelper(w)
	g.emitMapAssemblerMethods(w)
	g.emitKeyAssembler(w)
}
func (g unionBuilderGenerator) emitMapAssemblerChildTidyHelper(w io.Writer) {
	// This function attempts to clean up the state machine to acknolwedge child assembly finish.
	//  If the child was finished and we just collected it, return true and update state to maState_initial.
	//  Otherwise, if it wasn't done, return false;
	//   and the caller is almost certain to emit an error momentarily.
	// The function will only be called when the current state is maState_midValue.
	//  (In general, the idea is that if the user is doing things correctly,
	//   this function will only be called when the child is in fact finished.)
	// This is a *lot* simpler than the tidy behaviors needed for any of the other recursive kinds:
	//  unions don't allow either nullable nor optional members, so there's no need to process anything except Maybe_Value state,
	//  and the lack of need to consider nullable nor optionals also means we never need to worry about moving memory in the case of MaybeUsePtr modes.
	//  (FUTURE: this may get a bit more conditional if we support members that are of unit ype and have null as a representation.  Unsure how that would work out exactly, but should be possible.)
	// We don't bother to nil the child assembler's 'w' pointer: it's not necessary,
	//  because we'll never "share" 'cm' (as some systems, like maps and lists, do) or change its value (short of the whole assembler resetting),
	//   and therefore we should be able to rely on the child assembler to be reasonable and never start acting again after finish.
	//  (This *does* mean some care is required in the reset logic: we have to be absolutely sure that resetting propagates to all child assemblers,
	//   even if they're in other regions of the heap; otherwise, they might end up still holding actionable 'w' and 'm' pointers into bad times!)
	//  (If you want to compare this to the logic in struct assemblers: it's similar to how only children that don't have maybes need an active 'w' nil'ing;
	//   but the salient reason there isn't "because the don't have maybes"; it's "because they have a potentially-reused 'cm'".  We don't have the former; but we *also* don't have the latter, for other reasons.)
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) valueFinishTidy() bool {
			switch ma.cm {
			case schema.Maybe_Value:
				{{- /* nothing to do for memlayout=embedAll; the tag is already set and memory already in place. */ -}}
				{{- /* nothing to do for memlayout=interface either; same story, the values are already in place. */ -}}
				ma.state = maState_initial
				return true
			default:
				return false
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) emitMapAssemblerMethods(w io.Writer) {
	// DRY: I did an interesting thing here: the `switch ma.state` block remains textually identical to the one for structs,
	//  even though the branch by valueFinishTidy could jump directly to an error state.
	//   That same semantic error state gets checked separately a few lines later in a different mechanism.
	//    The later check is needed either way (the assembler needs to *keep* erroring if some derp calls AssembleEntry *again* after a previous call already did the tidy and got rejected),
	//     but we could arguably save a step there.  It would probably trade more assembly size for the cycles saved, too, though.
	//  Ah, tradeoffs.  I think the textually simple approach here is probably in fact the best.  But it could be done differently, yes.
	// Note that calling AssembleEntry again when it's not for the first entry *returns* an error; it doesn't panic.
	//  This is subtle but important: trying to add more data than is acceptable is a data mismatch, not a system misuse, and must error accordingly politely.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: AssembleEntry cannot be called when in the middle of assembling another key")
			case maState_expectValue:
				panic("invalid state: AssembleEntry cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: AssembleEntry cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on for the moment, but we'll still be erroring shortly.
			case maState_finished:
				panic("invalid state: AssembleEntry cannot be called on an assembler that's already finished")
			}
			if ma.ca != 0 {
				return nil, schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Detail: "cannot add another entry -- a union can only contain one thing!"}
			}
			{{- if .Type.Members }}
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member.Name }}":
				ma.state = maState_midValue
				ma.ca = {{ add $i 1 }}
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ma.w.tag = {{ add $i 1 }}
				ma.ca{{ add $i 1 }}.w = &ma.w.x{{ add $i 1 }}
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return &ma.ca{{ add $i 1 }}, nil
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				x := &_{{ $member | TypeSymbol}}{}
				ma.w.x = x
				if ma.ca{{ add $i 1 }} == nil {
					ma.ca{{ add $i 1 }} = &_{{ $member | TypeSymbol }}__Assembler{}
				}
				ma.ca{{ add $i 1 }}.w = x
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return ma.ca{{ add $i 1 }}, nil
				{{- end}}
			{{- end}}
			{{- end}}
			}
			return nil, schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Key:&_String{k}}
		}
	`, w, g.AdjCfg, g)

	// AssembleKey has a similar DRY note as the AssembleEntry above had.
	// One misfortune in this method: we may know that we're doomed to errors because the caller is trying to start a second entry,
	//  but we can't report it from this method: we have to sit on our tongue, slide to midKey state (even though we're doomed!),
	//   and let the keyAssembler return the error later.
	//    This sucks, but panicking wouldn't be correct (see remarks about error vs panic on the AssembleEntry method),
	//     and we don't want to make this call unchainable for everyone everywhere, either, so it can't be rewritten to have an immmediate error return.
	//    The transition to midKey state is particularly irritating because it means this assembler will be perma-wedged; but I see no alternative.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleKey() datamodel.NodeAssembler {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: AssembleKey cannot be called when in the middle of assembling another key")
			case maState_expectValue:
				panic("invalid state: AssembleKey cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: AssembleKey cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on for the moment, but we'll still be erroring shortly... or rather, the keyassembler will be.
			case maState_finished:
				panic("invalid state: AssembleKey cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midKey
			return (*_{{ .Type | TypeSymbol }}__KeyAssembler)(ma)
		}
	`, w, g.AdjCfg, g)

	// As with structs, the responsibilties of this are similar to AssembleEntry, but with some of the burden split into the key assembler (which should have acted earlier),
	//  and some of the logical continuity bounces through state in the form of 'ma.ca'.
	//  The potential to DRY up some of this should be plentiful, but it's a bit heady.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) AssembleValue() datamodel.NodeAssembler {
			switch ma.state {
			case maState_initial:
				panic("invalid state: AssembleValue cannot be called when no key is primed")
			case maState_midKey:
				panic("invalid state: AssembleValue cannot be called when in the middle of assembling a key")
			case maState_expectValue:
				// carry on
			case maState_midValue:
				panic("invalid state: AssembleValue cannot be called when in the middle of assembling another value")
			case maState_finished:
				panic("invalid state: AssembleValue cannot be called on an assembler that's already finished")
			}
			ma.state = maState_midValue
			switch ma.ca {
			{{- range $i, $member := .Type.Members }}
			case {{ add $i 1 }}:
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ma.ca{{ add $i 1 }}.w = &ma.w.x{{ add $i 1 }}
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return &ma.ca{{ add $i 1 }}
				{{- else if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "interface") }}
				x := &_{{ $member | TypeSymbol}}{}
				ma.w.x = x
				if ma.ca{{ add $i 1 }} == nil {
					ma.ca{{ add $i 1 }} = &_{{ $member | TypeSymbol }}__Assembler{}
				}
				ma.ca{{ add $i 1 }}.w = x
				ma.ca{{ add $i 1 }}.m = &ma.cm
				return ma.ca{{ add $i 1 }}
				{{- end}}
			{{- end}}
			default:
				panic("unreachable")
			}
		}
	`, w, g.AdjCfg, g)

	// Finish checks are nice and easy.  Is the maState in the right place now and was a 'ca' ever marked?
	//  If yes and yes, then together with the rules elsewhere, we must've processed and accepted exactly one entry; perfect.
	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) Finish() error {
			switch ma.state {
			case maState_initial:
				// carry on
			case maState_midKey:
				panic("invalid state: Finish cannot be called when in the middle of assembling a key")
			case maState_expectValue:
				panic("invalid state: Finish cannot be called when expecting start of value assembly")
			case maState_midValue:
				if !ma.valueFinishTidy() {
					panic("invalid state: Finish cannot be called when in the middle of assembling a value")
				} // if tidy success: carry on
			case maState_finished:
				panic("invalid state: Finish cannot be called on an assembler that's already finished")
			}
			if ma.ca == 0 {
				return schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Detail: "a union must have exactly one entry (not none)!"}
			}
			ma.state = maState_finished
			*ma.m = schema.Maybe_Value
			return nil
		}
	`, w, g.AdjCfg, g)

	doTemplate(`
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) KeyPrototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
		func (ma *_{{ .Type | TypeSymbol }}__Assembler) ValuePrototype(k string) datamodel.NodePrototype {
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member.Name }}":
				return _{{ $member | TypeSymbol }}__Prototype{}
			{{- end}}
			default:
				return nil
			}
		}
	`, w, g.AdjCfg, g)
}
func (g unionBuilderGenerator) emitKeyAssembler(w io.Writer) {
	doTemplate(`
		type _{{ .Type | TypeSymbol }}__KeyAssembler _{{ .Type | TypeSymbol }}__Assembler
	`, w, g.AdjCfg, g)
	stubs := mixins.StringAssemblerTraits{
		PkgName:       g.PkgName,
		TypeName:      g.TypeName + ".KeyAssembler",
		AppliedPrefix: "_" + g.AdjCfg.TypeSymbol(g.Type) + "__Key",
	}
	// This key assembler can disregard any idea of complex keys because we're fronting for a union!
	//  Union member names must be strings (and quite simple ones at that).
	stubs.EmitNodeAssemblerMethodBeginMap(w)
	stubs.EmitNodeAssemblerMethodBeginList(w)
	stubs.EmitNodeAssemblerMethodAssignNull(w)
	stubs.EmitNodeAssemblerMethodAssignBool(w)
	stubs.EmitNodeAssemblerMethodAssignInt(w)
	stubs.EmitNodeAssemblerMethodAssignFloat(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__KeyAssembler) AssignString(k string) error {
			if ka.state != maState_midKey {
				panic("misuse: KeyAssembler held beyond its valid lifetime")
			}
			if ka.ca != 0 {
				return schema.ErrNotUnionStructure{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Detail: "cannot add another entry -- a union can only contain one thing!"}
			}
			switch k {
			{{- range $i, $member := .Type.Members }}
			case "{{ $member.Name }}":
				ka.ca = {{ add $i 1 }}
				{{- if (eq (dot.AdjCfg.UnionMemlayout dot.Type) "embedAll") }}
				ka.w.tag = {{ add $i 1 }}
				{{- end}}
				ka.state = maState_expectValue
				return nil
			{{- end}}
			}
			return schema.ErrInvalidKey{TypeName:"{{ .PkgName }}.{{ .Type.Name }}", Key:&_String{k}} // TODO: error quality: ErrInvalidUnionDiscriminant ?
		}
	`, w, g.AdjCfg, g)
	stubs.EmitNodeAssemblerMethodAssignBytes(w)
	stubs.EmitNodeAssemblerMethodAssignLink(w)
	doTemplate(`
		func (ka *_{{ .Type | TypeSymbol }}__KeyAssembler) AssignNode(v datamodel.Node) error {
			if v2, err := v.AsString(); err != nil {
				return err
			} else {
				return ka.AssignString(v2)
			}
		}
		func (_{{ .Type | TypeSymbol }}__KeyAssembler) Prototype() datamodel.NodePrototype {
			return _String__Prototype{}
		}
	`, w, g.AdjCfg, g)
}
