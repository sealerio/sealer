go-ipld-prime
=============

`go-ipld-prime` is an implementation of the IPLD spec interfaces,
a batteries-included codec implementations of IPLD for CBOR and JSON,
and tooling for basic operations on IPLD objects (traversals, etc).



API
---

The API is split into several packages based on responsibly of the code.
The most central interfaces are the base package,
but you'll certainly need to import additional packages to get concrete implementations into action.

Roughly speaking, the core package interfaces are all about the IPLD Data Model;
the `codec/*` packages contain functions for parsing serial data into the IPLD Data Model,
and converting Data Model content back into serial formats;
the `traversal` package is an example of higher-order functions on the Data Model;
concrete `ipld.Node` implementations ready to use can be found in packages in the `node/*` directory;
and several additional packages contain advanced features such as IPLD Schemas.

(Because the codecs, as well as higher-order features like traversals, are
implemented in a separate package from the core interfaces or any of the Node implementations,
you can be sure they're not doing any funky "magic" -- all this stuff will work the same
if you want to write your own extensions, whether for new Node implementations
or new codecs, or new higher-order order functions!)

- `github.com/ipld/go-ipld-prime` -- imported as just `ipld` -- contains the core interfaces for IPLD.  The most important interfaces are `Node`, `NodeBuilder`, `Path`, and `Link`.
- `github.com/ipld/go-ipld-prime/node/basicnode` -- provides concrete implementations of `Node` and `NodeBuilder` which work for any kind of data, using unstructured memory.
- `github.com/ipld/go-ipld-prime/node/bindnode` -- provides concrete implementations of `Node` and `NodeBuilder` which store data in native golang structures, interacting with it via reflection.  Also supports IPLD Schemas!
- `github.com/ipld/go-ipld-prime/traversal` -- contains higher-order functions for traversing graphs of data easily.
- `github.com/ipld/go-ipld-prime/traversal/selector` -- contains selectors, which are sort of like regexps, but for trees and graphs of IPLD data!
- `github.com/ipld/go-ipld-prime/codec` -- parent package of all the codec implementations!
- `github.com/ipld/go-ipld-prime/codec/dagcbor` -- implementations of marshalling and unmarshalling as CBOR (a fast, binary serialization format).
- `github.com/ipld/go-ipld-prime/codec/dagjson` -- implementations of marshalling and unmarshalling as JSON (a popular human readable format).
- `github.com/ipld/go-ipld-prime/linking/cid` -- imported as `cidlink` -- provides concrete implementations of `Link` as a CID.  Also, the multicodec registry.
- `github.com/ipld/go-ipld-prime/schema` -- contains the `schema.Type` and `schema.TypedNode` interface declarations, which represent IPLD Schema type information.
- `github.com/ipld/go-ipld-prime/node/typed` -- provides concrete implementations of `schema.TypedNode` which decorate a basic `Node` at runtime to have additional features described by IPLD Schemas.


Getting Started
---------------

Let's say you want to create some data programmatically,
and then serialize it, or save it as [blocks].

You've got a ton of different options, depending on what golang convention you want to use:

- the `qp` package -- [example](https://pkg.go.dev/github.com/ipld/go-ipld-prime/fluent/qp#example-package)
- the `bindnode` system, if you want to use golang types -- [example](https://pkg.go.dev/github.com/ipld/go-ipld-prime/node/bindnode#example-Wrap-NoSchema), [example with schema](https://pkg.go.dev/github.com/ipld/go-ipld-prime/node/bindnode#example-Wrap-WithSchema)
- or the [`NodeBuilder`](https://pkg.go.dev/github.com/ipld/go-ipld-prime/datamodel#NodeBuilder) interfaces, raw (verbose; not recommended)
- or even some codegen systems!

Once you've got a Node full of data,
you can serialize it:

https://pkg.go.dev/github.com/ipld/go-ipld-prime#example-package-CreateDataAndMarshal

But probably you want to do more than that;
probably you want to store this data as a block,
and get a CID that links back to it.
For this you use `LinkSystem`:

https://pkg.go.dev/github.com/ipld/go-ipld-prime/linking#example-LinkSystem.Store

Hopefully these pointers give you some useful getting-started focal points.
The API docs should help from here on out.
We also highly recommend scanning the [godocs](https://pkg.go.dev/github.com/ipld/go-ipld-prime) for other pieces of example code, in various packages!

Let us know in [issues](https://github.com/ipld/go-ipld-prime/issues), [chat, or other community spaces](https://ipld.io/docs/intro/community/) if you need more help,
or have suggestions on how we can improve the getting-started experiences!



Other IPLD Libraries
--------------------

The IPLD specifications are designed to be language-agnostic.
Many implementations exist in a variety of languages.

For overall behaviors and specifications, refer to the IPLD website, or its source, in IPLD meta repo:
- https://ipld.io/
- https://github.com/ipld/ipld/
You should find specs in the `specs/` dir there,
human-friendly docs in the `docs/` dir,
and information about _why_ things are designed the way they are mostly in the `design/` directories.

There are also pages in the IPLD website specifically about golang IPLD libraries,
and your alternatives: https://ipld.io/libraries/golang/


### distinctions from go-ipld-interface&go-ipld-cbor

This library ("go ipld prime") is the current head of development for golang IPLD,
and we recommend new developments in golang be done using this library as the basis.

However, several other libraries exist in golang for working with IPLD data.
Most of these predate go-ipld-prime and no longer receive active development,
but since they do support a lot of other software, you may continue to seem them around for a while.
go-ipld-prime is generally **serially compatible** with these -- just like it is with IPLD libraries in other languages.

In terms of programmatic API and features, go-ipld-prime is a clean take on the IPLD interfaces,
and chose to address several design decisions very differently than older generation of libraries:

- **The Node interfaces map cleanly to the IPLD Data Model**;
- Many features known to be legacy are dropped;
- The Link implementations are purely CIDs (no "name" nor "size" properties);
- The Path implementations are provided in the same box;
- The JSON and CBOR implementations are provided in the same box;
- Several odd dependencies on blockstore and other interfaces that were closely coupled with IPFS are replaced by simpler, less-coupled interfaces;
- New features like IPLD Selectors are only available from go-ipld-prime;
- New features like ADLs (Advanced Data Layouts), which provide features like transparent sharding and indexing for large data, are only available from go-ipld-prime;
- Declarative transformations can be applied to IPLD data (defined in terms of the IPLD Data Model) using go-ipld-prime;
- and many other small refinements.

In particular, the clean and direct mapping of "Node" to concepts in the IPLD Data Model
ensures a much more consistent set of rules when working with go-ipld-prime data, regardless of which codecs are involved.
(Codec-specific embellishments and edge-cases were common in the previous generation of libraries.)
This clarity is also what provides the basis for features like Selectors, ADLs, and operations such as declarative transformations.

Many of these changes had been discussed for the other IPLD codebases as well,
but we chose clean break v2 as a more viable project-management path.
Both go-ipld-prime and these legacy libraries can co-exist on the same import path, and both refer to the same kinds of serial data.
Projects wishing to migrate can do so smoothly and at their leisure.

We now consider many of the earlier golang IPLD libraries to be defacto deprecated,
and you should expect new features *here*, rather than in those libraries.
(Those libraries still won't be going away anytime soon, but we really don't recomend new construction on them.)

### migrating

**For recommendations on where to start when migrating:**
see [README_migrationGuide](./README_migrationGuide.md).
That document will provide examples of which old concepts and API names map to which new APIs,
and should help set you on the right track.

### unixfsv1

Lots of people who hear about IPLD have heard about it through IPFS.
IPFS has IPLD-native APIs, but IPFS *also* makes heavy use of a specific system called "UnixFSv1",
so people often wonder if UnixFSv1 is supported in IPLD libraries.

The answer is "yes" -- but it's not part of the core.

UnixFSv1 is now treated as an [ADL](https://ipld.io/glossary/#adl),
and a go-ipld-prime compatible implementation can be found
in the [ipfs/go-unixfsnode](https://github.com/ipfs/go-unixfsnode) repo.

Additionally, the codec used in UnixFSv1 -- dag-pb --
can be found implemented in the [ipld/go-codec-dagpb](https://github.com/ipld/go-codec-dagpb) repo.

A "some assembly required" advisory may still be in effect for these pieces;
check the readmes in those repos for details on what they support.

The move to making UnixFSv1 a non-core system has been an arduous retrofit.
However, framing it as an ADL also provides many advantages:

- it demonstrates that ADLs as a plugin system _work_, and others can develop new systems in this pattern!
- it has made pathing over UnixFSv1 much more standard and well-defined
- this standardization means systems like [Selectors](https://ipld.io/glossary/#selectors) work naturally over UnixFSv1...
- ... which in turn means anything using them (ex: CAR export; graphsync; etc) can very easily be asked to produce a merkle-proof
  for a path over UnixFSv1 data, without requiring the querier to know about the internals.  Whew!

We hope users and developers alike will find value in how these systems are now layered.



Change Policy
-------------

The go-ipld-prime library is ready to use, and we value stability highly.

We make releases periodically.
However, using a commit hash to pin versions precisely when depending on this library is also perfectly acceptable.
(Only commit hashes on the master branch can be expected to persist, however;
depending on a commit hash in a branch is not recommended.  See [development branches](#development-branches).)

We maintain a [CHANGELOG](CHANGELOG.md)!
Please read it, when updating!

We do make reasonable attempts to minimize the degree of changes to the library which will create "breaking change" experiences for downstream consumers,
and we do document these in the changelog (often, even with specific migration instructions).
However, we do also still recommend running your own compile and test suites as a matter of course after updating.

You can help make developing this library easier by staying up-to-date as a downstream consumer!
When we do discover a need for API changes, we typically try to introduce the new API first,
and do at least one release tag in which the old API is deprecated (but not yet removed).
We will all be able to develop software faster, together, as an ecosystem,
if libraries can keep reasonably closely up-to-date with the most recent tags.


### Version Names

When a tag is made, version number steps in go-ipld-prime advance as follows:

1. the number bumps when the lead maintainer says it does.
2. even numbers should be easy upgrades; odd numbers may change things.
3. the version will start with `v0.` until further notice.

[This is WarpVer](https://gist.github.com/warpfork/98d2f4060c68a565e8ad18ea4814c25f).

These version numbers are provided as hints about what to expect,
but ultimately, you should always invoke your compiler and your tests to tell you about compatibility,
as well as read the [changelog](CHANGELOG.md).


### Updating

**Read the [CHANGELOG](CHANGELOG.md).**

Really, read it.  We put exact migration instructions in there, as much as possible.  Even outright scripts, when feasible.

An even-number release tag is usually made very shortly before an odd number tag,
so if you're cautious about absorbing changes, you should update to the even number first,
run all your tests, and *then* upgrade to the odd number.
Usually the step to the even number should go off without a hitch, but if you *do* get problems from advancing to an even number tag,
A) you can be pretty sure it's a bug, and B) you didn't have to edit a bunch of code before finding that out.


### Development branches

The following are norms you can expect of changes to this codebase, and the treatment of branches:

- The `master` branch will not be force-pushed.
    - (exceptional circumstances may exist, but such exceptions will only be considered valid for about as long after push as the "$N-second-rule" about dropped food).
    - Therefore, commit hashes on master are gold to link against.
- All other branches *can* be force-pushed.
    - Therefore, commit hashes not reachable from the master branch are inadvisable to link against.
- If it's on master, it's understood to be good, in as much as we can tell.
    - Changes and features don't get merged until their tests pass!
    - Packages of "alpha" developmental status may exist, and be more subject to change than other more finalized parts of the repo, but their self-tests will at least pass.
- Development proceeds -- both starting from and ending on -- the `master` branch.
    - There are no other long-running supported-but-not-master branches.
    - The existence of tags at any particular commit do not indicate that we will consider starting a long running and supported diverged branch from that point, nor start doing backports, etc.
