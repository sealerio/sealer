What's the deal with scalars, anyway?
=====================================

Two sorts of scalars
--------------------

There are two sorts of scalars that show up in codegen:

- 1: scalars that are just the plain kind (e.g. "string", not even named);
- 2: scalars that have named types.

Plain scalars can't have any special rules or semantics attached to them.

Named types with scalar kinds (aka a "typedef") **can** have additional rules and semantics attached to them.

Let's talk about named scalars first, because it's clearer that there's fun there.


### named scalars

Named scalars cause a type to be generated.
That type information is part of their identity (practically speaking: affects their definition of equality).

#### named scalars are never equal even if their contents are

It stands to reason that named scalars can't be freely interchanged.

If you have a schema:

```ipldsch
type Foo string
type Bar string
```

... then you'll get codegen output code with an exported type for each:

```go
type Foo struct{ x string }
/*...*/

type Bar struct{ x string }
/*...*/
```

... and clearly, `(Foo{"asdf"} == Bar{"asdf"}) == false`.

#### named scalars appear in specialized method argument types and return types

Just like any other named type, named scalars will appear in specialized methods
which are exported on codegen'd types.

For example, if you have a schema:

```ipldsch
type Foo string
type Bar string
type Foomp map {Foo:Bar}
```

... then you'll get codegen output code which includes a method on Foomp:

```go
func (x *Foomp) LookupByNode(k *Foo) (*Bar) { /*...*/ }
```

Such specialized methods are often much shorter, much more efficient to execute,
and involve much less error handling to use than their more generalized
counterparts on the `datamodel.Node` interface.

Note that when named scalars appear in the signitures of specialized methods,
they always appear as pointers.  They will never be `nil`, but there is still
a reason that pointers are used here, and it's based on performance.
(The details don't matter as a user, but: it means if those values need to be
regarded as the `datamodel.Node` interface again in the future, that boxing is
inexpensive since we already have a (heap-escaped long ago) pointer.
By contrast, copying by value in more places is likely to result in more
heap escapes and thus additional undesirable new allocation costs in the
(entirely common!) case that the values end up handled as `datamodel.Node` later.)

#### named scalars have a specialized method which unboxes them to a native primitive unconditionally

Every named scalar type as a specialized unbox method corresponding to its kind.

For example, for a `type Foo string`, there will be a `func (f Foo) String() string` method
(in addition to the `func (f Foo) AsString() (string, error)` method,
which does the same thing but is stuck presenting an error due to interface conformance even though we know that it's statically impossible).

#### named scalars can have additional methods attached to them

It's possible for users of codegen to attach additional methods to the types
generated for a named scalar.

This can be either done for purely aesthetic/ergonomic purposes particular
to the user's exact product, or, as part of some extended library features.
For example, we plan support extended features like "validation" methods
via detecting when a user adds a `Valdiate() error` method to a generated type.


### plain scalars

Plain scalars also cause a type to be generated;
one type for each kind in the Data Model is sufficient.

Plain scalars show up in codegen output packages almost exactly as if
there was a short preamble in every schema:

```ipldsch
type Int int
type Bool bool
type Float float
type String string
type Bytes bytes
```

#### note about schema syntax

There's an issue about capitalization that's somewhat unresolved in schemas:
namely, is `type Fwee struct { someField string }` allowed, or a parse error?

This syntax is questionable because it means some of the scalar kind identifier
keywords are allowed in the same place as type names,
and it's potentially confusing because when we come to interacting with the
generated output code in golang, we still have `String`-with-a-capital-S
as a type identifier.

At any rate, it seems clear that you can mentally capitalize the 's'
at any time you see this debatable syntax.

(We should resolve this issue in the specs, which are in the `datamodel.specs` repo.)

#### plain scalars appear in specialized method argument types and return types

This is the same story as for named scalars.

For example, if you have a schema:

```ipldsch
type Foomp map {String:String}
```

... then you'll get codegen output code which includes a method on Foomp:

```go
func (x *Foomp) LookupByNode(k String) (String) { /*...*/ }
```

(The exact symbols involved and whether or not they're pointers may vary.)

The type might carry less semantic information than it does when a
named scalar shows up in the same position, but we still use a generated
type (and a pointer) here for two reasons: first of all, and more simply,
consistency; but secondly, for the same performance reasons as applied
to named scalars (if we need to treat this value as an `datamodel.Node` again
in the future, it's much better if we already have a heap pointer rather
than a bare primitive value (`runtime.convT*` functions are often not your
favorite thing to see in a pprof flamegraph)).

(FUTURE: this is still worth review.  We might actually want to use
bare primitives in a lot of these cases, because surely, if you're about
to want to treat something as an `datamodel.Node` again, then you can use the
generalized methods conforming to `datamodel.Node` which already yield that...?
We'll get more information and impressions about this after trying to use
codegen in bulk (especially the specialized methods).)

#### plain scalars do not allow additional method attachedments

While we can't *stop* developers from modifying the source code emitted by codegen,
adding a method to any of the plain scalars is intensely discouraged.
Nothing sensible or good can come of trying to attach a "Validate" method
to something like the `String` type.  Don't do it.


Code reuse for plain scalars
----------------------------

We *always* need some type that can contain a plain scalar while also
implementing all the `datamodel.Node` methods.  Even if we didn't export it
or show it in any method signitures anywhere at all, we'd *still* need it
for internal implementation of other types, because it's important those
types be able to return a pointer to their fields in their implements of
the `datamodel.Node` contract (otherwise, they'd be terribly slow and alloc-heavy).

### can we reuse another package's plain scalars?

Since there's no functional difference between the plain scalars in a schema
and the scalars implementation from another package that's untyped in the first place,
can we reuse some code from an untyped package in codegen output packages?

No.

(Or: "maybe, conditionally, and it would have a lot of caveats and make the
untyped package we try to hitch a ride on become significantly weirder, so...
it's probably not worth it".)

The reason to desire this so there's less (admittedly quite duplicative) code
in the package emitted by using codegen.

However, there are *many* "cons" which outweight that single "pro":

- This would require the untyped package to export their concrete implementation types.
	- This is the *only* reason those implementation types would need to be exported, which is a concerning smell all by itself.
	- In the case of we consider using the 'basicnode' package in particular:
		- Exporting those types allows creation by casting, which exposes an API surface that's not conventional (nor necessarily even possible) for other packages, and will thus be likely to create confusion as well as create multiple ways of doing things which will make refactors harder.
			- We don't like allowing casting for creating values in general for reasons explored well in the go-cid refactors to use wrapper structs: if casting is possible, it's far too easy for an end-user to write shoddy code which dodges all constructors and validation logic.
		- Exporting those types allows unboxing by casting, which again exposes an API surface that's not conventional (nor necessarily even possible) for other packages, and will thus be likely to create confusion as well as create multiple ways of doing things which will make refactors harder.
			- Since we're talking about scalars and they're essentially copy-by-value (except for bytes -- but we give up and rely on "lawful" code for those anyway, since defensive copies are completely nonviable in performance terms), this doesn't create incorrectness issues... but it's still not *good*.
			- Note that while casting to concrete types exported by the output package of codegen is considered acceptable, this is a different beast: you still can't get the raw content out without using at least one more unboxing method; and, if you're casting or doing a type switch with type in a codegen package, it should already instantly be clear that your code is no longer general-purpose, and this will surprise no one.
		- ...And while the above two are true only because the implmentation is by typedefs and they could be fixed by using a wrapper struct... that fix would have exactly the effect of making reuse impossible anyway, since the field in that wrapper struct would need to be unexported (otherwise, immutability would then in turn trivially shatter).
		- The implementation of the scalar for link kinds can't be reused anyway (it *does* use a wrapper struct already, and needs to; type aliases on an interface don't permit adding methods), adding yet more inconsistency and jagged edges to the picture.
		- The "more unnecessarily(-for-end-user-perspectives) exported symbols" code smell counts about 10x as hard for this package in particular, since it's often one of the first ones a newcomer to this library will see: there shouldn't be weird designs with elaborate and far away justifications poking up here.
- Reusing concrete types between packages makes it more likely uncautious users could write code that uses native equality on scalars and get away with it *sometimes*.  Since this is still incorrect and would sometimes fail in fully general code, it's better if code like this flunks out as early as possible, which results in a better ecosystem overall.
- We like it when error messages can include a type name.  It's marginally better for that to be something like "gendemo.String" ('gendemo' being consistent with whatever the rest of the package also says) than just bare "string".

There are also a few bits that aren't entirely known (at least, at the time of this writing):
namely, how 'any' types are going to be handled in codegen.
Probably, though, the answer is: it's just treated as 'datamodel.Node',
and the codegen package doesn't export *any* more types which regard this situation because that's already sufficient.

Long story short?  It's better to have plain scalar types in codegen output,
even if they look somewhat duplicative,
because trying to do anything fancier either fails outright
or spawns ridiculously detailed epicycles of complexity.
Emitting the plain scalar types in codegen output
is *more consistent* in almost every way,
will generate less cognitive load for users,
and just plain *works unconditionally*.
