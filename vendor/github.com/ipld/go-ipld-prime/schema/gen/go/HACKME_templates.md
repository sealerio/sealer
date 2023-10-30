Notes about how Templates are used in this package
==================================================

This package makes heavy use of go's `text/template`.

Some of it is not pretty.  But beware of trying to trade elegance for legibility.

An overview of the choices that got us here:

### string templating is fine

String templating for code is fine, actually.

An alternative would be to use the golang AST packages.
While those are nice... for our purposes, they're a bit verbose.
It's not necessarily helpful to have the visual distance between our
generation code and the actual result _increase_ (at least, without a good reason).

### don't make indirections around uncomplicated munges

Don't make indirections that aren't needed for simple string operations.
(This goes for both the template funcmap, or in the form of reusable templates.)

One very common situation is that for some type `T`, there's a related thing to be generated
called `T__Foo`, where "Foo" is some kind of modifier that is completely predictable
and not user-selected nor configurable.

In this situation: simply conjoining `{{ .T }}__Foo` as string in the template is fine!
Don't turn it into a shell game with functions that make the reader jump around more to see what's happening.
(Even when refactoring, it's easy enough to use simple string replace on these patterns;
extracting a function to enable "changing it in one place" doesn't really add much value.)

If there's something more complicated going on, or it's user-configurable?
Fine, get more functions involved.  But be judicious about it.

(An earlier draft of the code introduced a new method for each modifier form in the example above.
The result was... just repeating the modifiers in the function name in the template!
It produced more code, yet no additional flexibility.
It is advisable to resist making this mistake again ;))

### maintain distinction between **symbol** versus **name**

- **symbol** is what is used in type and function names in code.
- **name** is the string that comes from the schema; it is never modified nor overridable.

Symbols can change based on adjunct config.
(In theory, they could also be munged to avoid collisions with language constructs
or favor certain library conventions... however, this is currently not automatic.)

Names can't change.  They're _exactly_ what was written in in the Schema.

Types and functions and most of the _code_ part of codegen use Symbols.
Error messages, reflectable type information, etc, use Names.

Symbols may also need to be _generated_ for types that don't have names.
For example, `type Foo struct { field {String:Int} }` might result in codegen
creating *two* symbols: `Foo`, and `Map__String__Int`.

One way to check that the goal is being satisfied is to consider that
someone just experiencing error messages from a program should not end up exposed to any information about:

- what language the program is written in,
- or which codegen tool was used (or even *if* a codegen tool was used),
- or what adjunct config was present when codegen was performed.

(n.b. this consideration does not include errors with stack traces -- then
of course you will unavoidably see symbols appear.)

### anything that is configurable goes through adjunctCfg; adjunctCfg is accessed via the template funcmap

For example, all Symbol processing all pipes through an adjunct configuration object.
We make this available in the templates via a funcmap so it's available context-free as a nice tidy pipe syntax.

### there are several kinds of reuse

(Oh, boy.)

It may be wise to read the ["values we can balance"](./HACKME_tradeoffs.md#values-we-can-balance) document before continuing.
There's also a _seventh_ tradeoff to consider, in addition to those from that document:
how much reuse there is in our _template_ code, and (_eighth!_) how _readable_ and maintainable the template code is.
Also, (_ninth!!_) how _fast_ the template code is.

We've generally favored *all* of the priorities for the output code (speed, size, allocation count, etc) over the niceness of the codegen.
We've also _completely_ disregarded speed of the template code (it's always going to be "fast enough"; you don't run this in a hot loop!).
When there's a tension between readability and reuse, we've often favored readability.
That means sometimes text is outright duplicated, if it seemed like extracting it would make it harder to read the template.

Here's what we've ended up with:

- There are mixins from the `node/mixins` package, which save some amount of code and standardize some behaviors.
	- These end up in the final result code.
	- It doesn't save *much* code, and they generally don't save *any* binary size (because it all gets inlined).
	- The consistency is the main virtue of these.
	- These are used mainly for error handling (specifically, returning of errors for methods called on nodes that have the wrong kind for that method).
- There are mixins from the `schema/gen/go/mixins` package.
	- These are in the template code only -- they don't show up in the final result code.
	- These attempt to make it easier to create new 'generator' types.  (There's a 'generator' type for each kind-plus-representation.)
	- They only attempt to take away some boilerplate, and you don't _have_ to use them.
- There are functions in the template funcmap.
	- ... not many of them, though.
- There's the idea of using associated templates (templates that are invoked by other templates).
	- There's currently none of this in use.  Might it be helpful?  Maybe.
- There are functions which apply well-known templates to a generator.
	- These compose at the golang level, so it's easy to have the compiler check that they're all in order without running them (unlike templates, which have to be exercised in order to detect even basic problems like "does this named template exist").
	- Many of these assume some pattern of methods on the generator objects.  (Not of all these are super well documented.)
	- Generators usually call out to one or more of these from within the methods that their interface requires them to have.
- The generator types themselves are usually split into two parts: the mechanisms for type-plus-repr, and just mechanisms for the type.
	- The mechanisms for the type alone aren't actually a full generator.  The type-plus-repr thing just embeds the type-level semantics in itself.

*Mostly*, it's general aim has been to keep relatively close to the structure of the code being generated.
When reading a generator, one generally has to do *zero* or *one* jump-to-definition in order to see the fulltext of a template -- no more than that.
(And so far, all those jump-to-definition lookups are on _go code_, not inside the template -- so an IDE can help you.)

By example: if there are things which turn out common between _representation kinds_,
those will probably end up in a function containing a well-known template,
and that will end up being called from the generator type in one of the functions its required to have per its interface contract.

This all probably has plenty of room for improvement!
But know you know the reasoning that got things to be in the shape they are.
