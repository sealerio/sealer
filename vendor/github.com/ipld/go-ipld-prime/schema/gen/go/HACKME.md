hacking gengo
=============

What the heck?
--------------

We're doing code generation.

The name of the game is "keep it simple".
Most of this is implemented as string templating.
No, we didn't use the Go AST system.  We could have; we didn't.
Implementing this as string templating seemed easier to mentally model,
and the additional value provided by use of AST libraries seems minimal
since we feed the outputs into a compiler for verification immediately anyway.

Some things seem significantly redundant.
That's probably because they are.
In general, if there's a choice between apparent redundancy in the generator itself
versus almost any other tradeoff which affects the outputs, we prioritize the outputs.
(This may be especially noticable when it comes to error messages: we emit a lot
of them... while making sure they contain very specific references.  This leads
to some seemingly redundant code, but good error messages are worth it.)

See [README_behaviors](README_behaviors.md) for notes about the behaviors of the code output by the generator;
this document is about the generator code itself and the design thereof.


Entrypoints
-----------

The most important intefaces are all in [`generators.go`](generators.go).

The function you're most likely looking for that "does the thing" is the
`Generate(outputPath string, pkgName string, schema.TypeSystem, *AdjunctCfg)` method,
which can be found in the [`generate.go`](generate.go) file.
You can take any of the functions inside of that and use them as well,
if you want more granular control over what content ends up in which files.

The eventual plan is be able to drive this whole apparatus around via a CLI
which consumes IPLD Schema files.
Implementing this can come after more of the core is done.
(Seealso the `schema/tmpBuilders.go` file a couple directories up for why
this is currently filed as nontrivial/do-later.)


Organization
------------

### How many things are generated, anyway?

There are roughly *seven* categories of API to generate per type:

- 1: the readonly thing a native caller uses
- 2: the builder thing a native caller uses
- 3: the readonly typed node
- 4: the builder/assembler for typed node
- 5: the readonly representation node
- 6: the builder/assembler via representation
- 7: and a maybe wrapper

(And these are just the ones nominally visible in the exported API surface!
There are several more concrete types than this implied by some parts of that list,
such as iterators for the nodes, internal parts of builders, and so forth.)

These numbers will be used to describe some further organization.

### How are the generator components grouped?

There are three noteworthy types of generator internals:

- `TypeGenerator`
- `NodeGenerator`
- `NodebuilderGenerator`

The first one is where you start; the latter two do double duty for each type.

Exported types for purpose 1, 2, 3, and 7 are emitted from `TypeGenerator` (3 from the embedded `NodeGenerator`).

The exported type for purpose 5 is emitted from another `NodeGenerator` instance.

The exported types for purposes 4 and 6 are emitted from two distinct `NodebuilderGenerator` instances.

For every variation in type kind and representation strategy for that type kind,
one type implementing `TypeGenerator` is composed, and it has functions which
yield all the other interfaces for addressing the various purposes.

### How are files and their contents grouped?

Most of the files in this package are following a pattern:

- for each kind:
	- `gen{Kind}.go` -- has emitters for the native type parts (1, 2, 7) and type-level node behaviors (3, 4).
	- for each representation that kind can have:
		- `gen{Kind}Repr{ReprStrat}.go` -- has emitters for (5, 6).

A `mixins` sub-package contains some code which is used and embedded in the generators in this package.
These features are mostly per-kind -- representation kind, not type-level kind.
For example, you'll see "map" behaviors from the mixins package added to "struct" generators.

### What are all these abbreviations?

See [HACKME_abbrevs.md](HACKME_abbrevs.md).

### Code architecture

See [HACKME_tradeoffs.md](HACKME_tradeoffs.md) for an overview of tradeoffs,
and which priorities we selected in this package.
(There are *many* tradeoffs.)

See [HACKME_memorylayout.md](HACKME_memorylayout.md) for a (large) amount of
exposition on how this code is designed in order to be allocation-avoidant
and fast in general.

See [HACKME_templates.md](HACKME_templates.md) for some overview on how we've
used templates, and what forms of reuse and abstraction there are.

See [HACKME_scalars.md](HACKME_scalars.md) for some discussion of scalars
and (why we generate more of them than you might expect).

See [HACKME_maybe.md](HACKME_maybe.md) for notes how how the 'maybe' feature
(how we describe `nullable` and `optional` schema features in generated golang code)
has evolved.


Testing
-------

See [HACKME_testing.md](HACKME_testing.md) for some details about how this works.

In general, try to copy some of the existing tests and get things to suit.

Be advised that we use the golang plugin feature, and that has some additional
requirements of your development environment than is usual in golang.
(Namely, you have to be on linux and you have to have a c compiler!)

