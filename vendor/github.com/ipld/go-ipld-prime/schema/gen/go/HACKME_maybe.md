How do maybe/nullable/optional work?
====================================

(No, this document is not about things that we should "maybe" hack on.
It's about the feature we use to describe `nullable` and `optional` fields
in generated golang code.)

background
----------

You'll need to understand what the `nullable` and `optional` modifiers in IPLD schemas mean.
The https://specs.datamodel.io/ site has more content about that.

### how this works outside of schemas

There are concepts of null and of absent present in the core `Node` and `NodeAssembler` interfaces.
`Node` specifies `IsNull() bool` and `IsAbsent() bool` predicates;
and `NodeAssembler` specifies an `AssignNull` function.

There are also singleton values available called `datamodel.Null` and `datamodel.Absent`
which report true for `IsNull` and `IsAbsent`, respectively.
These singletons can be used by an function that need to return a null or absence indicator.

There's really no reason for any package full of `Node` implementations need to make their own types for these values,
since the singletons are always fine to use.
However, there's also nothing stopping a `Node` implementation from doing interesting
custom internal memory layouts to describe whether they contain nulls, etc --
and there's nothing particularly blessed about the `datamodel.Null` singleton.
Any value reporting `IsNull` to be `true` must be treated indistinguishably from `datamodel.Null`.

This indistinguishability is bidirectional.
For example, if you have some `myFancyNodeType`, and it answers `IsNull` as `true`,
and you insert this into a `basicnode.Map`, then ask for that value back from the map later...
you're very likely to get `datamodel.Null`, and not your concrete value of `myFancyNodeType` back again.
(This contract is important because some node implementations may compress
the concept of null into a bitmask, or otherwise similarly optimize things internally.)

#### null

The concept of "null" has a Kind in the IPLD Data Model.
It's implemented by the `datamodel.nullNode` type (which has no fields -- it's a "unit" type),
and is exposed as the `datamodel.Null` singleton value.

(More generally, `datamodel.Node` can be null by having its `Kind()` method return `datamodel.Kind_Null`,
and having the `IsNull()` method return `true`.
However, most code prefers to return the `datamodel.Null` singleton value whenever it can.)

Null values can be easily produced: the `AssignNull()` method on `datamodel.NodeAssembler` produces nulls;
and many codecs have some concept of null, meaning deserialization can produce them.

Null values work essentially the same way in both the plain Data Model and when working with Schemas.

#### absent

There's also a concept of "absent".
"Absent" is separate and distinct from the concept of "null" -- null is still a _value_; absent just means _nothing there_.

(Those familiar with javascript might note that javascript also has concepts of "null" versus "undefined".
It's the same idea -- we just call it "absent" instead of "undefined".)

Absent is implemented by the `datamodel.absentNode` type (which has no fields -- it's a "unit" type),
and is exposed as the `datamodel.Absent` singleton value.

(More generally, an `datamodel.Node` can describe itself as containing "absent" by having the `IsAbsent()` method return `true`.
(The `Kind()` method still returns `datamodel.Kind_Null`, for lack of better option.)
However, most code prefers to return the `datamodel.Absent` singleton value whenever it can.)

Absent values aren't really used at the Data Model level.
If you ask for a map key that isn't present in the map, the lookup method will return `nil` and `ErrNotExists`.

Absent values *do* show up at the Schema level, however.
Specifically, in structs: a struct can have a field which is `optional`,
one of the values such an optional field may report itself as having is `datamodel.Absent`.
This represents when a value *wasn't present* in the serialized form of the struct,
even though the schema lets us know that it could be, and that it's part of the struct's type.
(Accordingly, no `ErrNotExists` is returned for a lookup of that field --
the field is always considered to _exist_... the value is just _absent_.)
Iterators will also return the field name key, together with `datamodel.Absent` as the value.

However, absent values can't really be *created*.
There's no such thing as an `AssignAbsent` or `AssignAbsent` method on the `datamodel.NodeAssembler` interface.
Codecs similarly can't produce absent as a value (obviously -- codecs work over `datamodel.NodeAssembler`, so how could they?).
Absent values are just produced by implication, when a field is defined, but its value isn't set.

Despite absent values not being used or produced at the Data Model, we still have methods like `IsAbsent` specified
as part of the `datamodel.Node` interface so that it's possible to write code which is generic over
either plain Data Model or Schema data while using just that interface.

### the above is all regarding generic interfaces

As long as we're talking about the `datamodel.Node` _interface_,
we talk about the `datamodel.Null` and `datamodel.Absent` singletons, and their contracts in terms of the interface.

(Part of the reason this works is because an interface, in golang,
comes in two parts: a pointer to the typeinfo of the inhabitant value,
and a pointer to the value itself.
This means anywhere we have an `datamodel.Node` return type, we can toss `datamodel.Null`
or `datamodel.Absent` into it with no additional overhead!)

When we talk about concrete types, rather than the `datamodel.Node` _interface_ --
as we're going to, in codegen -- it's a different scenario.
We can't just return `datamodel.Null` pointers for a `genresult.Foo` value;
if `genresult.Foo` is a concrete type, that's just flat out a compile error.

So what shall we do?

We introduce the "maybe" types.



the maybe types
---------------

The general rule of "return `datamodel.Null` whenever you have a null value"
holds up only as long as our API is returning monomorphized `datamodel.Node` interfaces --
in that situation, `datamodel.Null` fits within `datamodel.Node`, and there's no trouble.

This doesn't hold up when we get to codegen.
Or rather, more specifically, it even holds up for codegen...
as long as we're still returning monomorphized `datamodel.Node` interfaces (and a decent amount of the API surface still does so).
At the moment we want to return a concrete native type, it breaks.

We call methods created by codegen that use specific types
(e.g., methods that you _couldn't have_ without codegen)
"speciated" methods.  And we do want them!

So we have to decide how to handle null and absent for these speciated methods.

### goals of the maybe types

There are a couple of things we want to accomplish with the maybe types:

- Be able to have speciated methods that return a specific type (for doc, editor autocomplete, etc purposes).
- Be able to have speciated methods that return specific *concrete* type (i.e. not only do we want to be more specific than `datamodel.Node`, we don't want an interface _at all_ -- so that the compiler can do inlining and optimization and so forth).
- Make reading and writing code that uses speciated methods and handles nullable or optional fields be reasonably ergonomic (and as always, this may vary by "taste").

And we'll consider one more fourth, bonus goal:

- It would be nice if the maybe types can clearly discuss whether the type is `(null|value)` vs `(absent|value)` vs `(absent|null|value)`, because this would let the golang compiler help check more of our logical correctness in code written using optionals and nullables.

### there is only one type generated for each maybe

For every type generated, there is one maybe type also generated.
(At least this much is clearly necesary to satisfy the goals about "specific types".)

This means *we dropped the bonus goal* above.
Making `(null|value)` vs `(absent|value)` vs `(absent|null|value)` distinguishable to the golang compiler
would require three *additional* generated types (for obvious reasons) for each type specified by the Schema.
We decided that's simply too onerous.

(A different codegen project could certainly make a different choice here, though.)

### the symbol for maybe types

For some type named `T` generated into a package named `gen`...

- the main type symbol is `gen.T`;
- the maybe for that type is `gen.MaybeT`;

Beware that this may spell trouble if your schema contains any types
with names starting in "Maybe".
(You can use adjunct config to change symbols for those types, if necessary.)

(There are also internal symbols for the same things,
but these are prefixed in such a way as to make collision not a concern.)

### maybe types don't implement the full Node interface

The "maybe" types don't implement the full `datamodel.Node` interface.
They could have!  They don't.

Arguments that went in favor of implementing `Node`:

- generally "seem fine"
- certainly makes sense to be able to 'IsNull' on it like any other Node.
- if in practice the maybe is embeded, we can return an internal pointer to it just fine, so there's no obvious runtime perf reason not to.

Arguments against:

- it's another type with a ton of methods.  or two, or four.
	- may increase the binary size.  possibly by a significant constant multiplier.
	- definitely increases the gsloc size, significantly.
- would it have a 'Type' accessor on it?
	- if so, what does it say?
- simply not sure how useful this is!
	- istm one will often either be passing the MaybeT to other speciated functions, or, fairly immediately de-maybing it.
		- if this is true, the number of times anyone wants to treat it as a Node in practice are near zero.
- does this imply the existence of a _MaybeT__Assembler type, as well?
	- binary and gsloc size still drifting up; this needs to justify itself and provide value to be worth it.
	- what would be the expected behavior of handing a _MaybeT__Assembler to something like unmarshalling?
		- if you have a null in the root, you can describe this with a kinded union, and probably would be better off for it.
		- if you have can absent value in the root of a document you're unmarshalling... what?  That's called "EOF".
	- does a _MaybeT__Assembler show up usefully in the middle of a tree?
		- it does not!  there's always a _P_ValueAssembler type involved there anyway (this is needed for parent state machine purposes), and it largely delegates to the _T__Assembler, but is already a perfect position to add on the "maybe" semantics if the P type has them for its children.

The arguments against carried the day.

### the maybe type is emebbedable

It's important that the "maybe" types be embeddable, for all the same reasons that
[we normally want embeddable types](./HACKME_memorylayout.md#embed-by-default).

It's interesting to consider the alternatives, though:

We could've bitpacked the isAbsent or isNull flags for a field into one word at the top of a struct, for example.
But, there are numerous drawbacks to this:

- the complexity of this is high.
- it would be exposed to anyone who writes addntl code in-package, which is asking for errors.
- the only thing this buys us is *slightly* less resident memory size.
	- and long story short: if you look at how many other programming language do this, pareto-wise, no one in the world at large appears to care.
- it only applies to structs!  maps or lists would require yet more custom bitpacking of a different arrangement.

If someone wants to do another codegen project someday, or make PRs to this one, which does choose bitpacking,
it would probably be neat.  It's just a lot of effort for a payout that doesn't seem to often be worth it.

(We also ended up using pointers to a field with a `schema.Maybe` type _heavily_
in the internals of our codegen outputs, in order to let child and parent assemblers coordinate.
Rebuilding this to work with a bitpacking alignment and yet still be composable enough to do its job... uufdah.  Tricky.
It might be possible to use the current system in the assembler state, but flip it bitpack in the resulting immutable nodes,
and thereby get the best of both worlds.  If you who reads this is enthusiastic, feel free to explore it.)

### ...but the user is only exposed to the pointer form

This is the same story as for the main types: it's covered in
[unexported implementations, exported aliases](./HACKME_memorylayout.md#unexported-implementations-exported-aliases).

Genenerally, this "shielded" type means you can only have a MaybeT with valid contents,
because no one can ever produce the uninitialized "zero" value of the type.
This means there's no "invalid" state which can kick you in the shins at runtime,
and we generally regard that as a good thing.

It also just keeps things syntactically simple.
One always refers to "MaybeT"; never with a star.

### whether or not the maybe's inhabitant type is embedded is based on adjunct config

Although the maybe type itself is embeddable, its _inhabitant_ may be
either embedded in the maybe type or be a pointer, at your option.

This is clearest to explain in code: you can have either:

```go
type MaybeFoo struct {
	m schema.Maybe // enum bit for present|absent|null
	v Foo          // the inhabitant (here, embedded)
}
```

or:

```go
type MaybeFoo struct {
	m schema.Maybe // enum bit for present|absent|null
	v *Foo         // the inhabitant (here, a pointer!)
}
```

(Yes, we're talking about a one-character difference in the code.)

Which of these two forms is generated can be selected by adjunct config.
("Adjunct" config just means: it's not part of the schema; it's part of the
config for this codegen tool.)

There are advantages to each:

- the embedded form is ([as usual](./HACKME_memorylayout.md#embed-by-default)), faster for workloads where the value is usually present (it provokes fewer allocations).
- the pointer form may use less memory when the value is absent; it works for cyclic structures; and if assigning a whole subtree at once, it allows faster assignment.

Also, for cyclic structures, such as `type Foo {String:nullable Foo}`, or `type Bar struct{ recurse optional Bar }`, the pointer form is *required*.
(Otherwise... how big of a slab of memory would we be allocating?  Infinite?  Nope; compile error.)

By default, we generate the pointer form.
However, your application may experience significant performance improvements by selectively using the embed form.
Check it out and tune for what's right for your application.

(FUTURE: we should make more clever defaults: it's reasonable to default to embed form for any type that is of scalar kind.)



implementation detail notes
---------------------------

### how state machines and maybes work

Assemblers for recursive stuff have state machines that are used to insure
orderly transitions between each key and value assembly,
and that a complete entry has been assembled before the next entry or the finish.
(For example, you can't go key-then-key in a map,
nor start a value and then start another value before finishing the first one in a list,
nor finish a map when you've just inserted a key and no value, and so forth.)

One part of this is straightforward: we simply implement state machines,
using bog-standard patterns around a typed uint and logical transition guards
in all the relevant functions.  Done and done.  Except...

How do child assemblers signal to their parent that they've become finished?
Theoretically, easy; in practice, to work efficiently...
This poses a bit of an implementation challenge.

One obvious solution is to put a callback field in assemblers, and have
the parent assembler supply the child assembler with a callback that can
update the parent's state machine when the child becomes finished.
This is logically correct, but practically, problematic and Not Fast:
it requires generating a closure of some kind which composes the function
pointer with the pointer to that parent assembler: and since this is two words
of memory, it implies an allocation and (unfortunately) a heap escape.
An allocation per child key and value in a recursive structure is unacceptable;
we want to set a _much_ higher bar for performance here.

So, we move on to less obvious solutions: we're all in the same package here,
so we can twiddle the bits of our neighboring structures quite directly, yes?
What if we just have assemblers contain pointers to a state machine uint,
and they do a fixed-value compare-and-swap when they're done?
This is terrifyingly direct and has no abstractions, yes indeed: but
we do generally assume control all the code in this package for any of our
correctness constraints, so this is in-bounds (if admittedly uncomfortable).

Now let's combine that with one more concern: nullables.  When an assembler
is not at the root of a document, it may need to accept null values.
We could do this by generating distinct assembler types for use in positions
where nulls are allowed; but though such an approach would work, it is bulky.
We'd much rather be able to reuse assembler types in either scenario.

So, let's have assemblers contain two pointers:
the already-familiar 'w' pointer, and also an 'm' pointer.
The 'm' pointer effectively communicates up whether the child has become finished
when it becomes either 'Maybe_Null' or 'Maybe_Value'.

We add a few new states to the 'm' value, and use it to hint in both directions:
assemblers will assume nulls are not an acceptable transition *unless* the 'm'
value comes initialized with a hint that we are in a situation where they work.

The costs here are "some": it's another pointer indirection and memory set.
However, compared to the alternatives, it's pretty good: versus an allocation
(in the callback approach), this is a huge win; and we're even pretty safe to
bet that that pointer indirection is going to land in a cache line already hot.

You can find the additional magic consts crammed into `schema.Maybe` fields
for this statekeeping during assembly defined in the "minima" file in codegen output.
They are named `midvalue` and `allowNull`.



this could have been different
------------------------------

There are many ways this design could've been different:

### we could have every maybe type implement Node

As already discussed above, it would cause a lot of extra boilerplate methods,
increasing both the generated code source size and binary size;
but on the plus side, it would've been in some ways arguably more consistent.

We didn't.

### we could've generated three maybes per type

Already discussed above.

We didn't.

### we could've designed schemas differently

A lot of the twists of the design originate from the fact that both `optional`
and `nullable` are both rather special as well as very contextual in IPLD Schemas
(e.g., `optional` is only permitted in a very few special places in a schema).
If we had built a very different type system, maybe things would come out differently.

Some of this has some exploration in some gists:

- https://gist.github.com/warpfork/9dd8b68deff2b90f96167c900ea31eec#dubious-soln-drop-nullable-completely-make-inline-anonymous-union-syntax-instead
- https://gist.github.com/warpfork/9dd8b68deff2b90f96167c900ea31eec#soln-change-how-schemas-regard-nullable-and-optional
- https://gist.github.com/warpfork/9dd8b68deff2b90f96167c900ea31eec#soln-support-absent-as-a-discriminator-in-kinded-unions

But suffice to say, that's a very big topic.

Optionals and nullables are the way they are because they seemed like useful
concepts for describing the structure of data which has serial forms;
how they map onto any particular programming language (such as Go) was a secondary concern.
This design for a golang library is trying to do its best within that.

### we could've done X with techinque Y

Probably, yes :)

This is just one implementation of codegen for Golang for IPLD Schemas.
Competing implementations that make different choices are absolutely welcome :)



