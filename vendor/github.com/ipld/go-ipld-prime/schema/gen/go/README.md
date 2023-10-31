gengo
=====

This package contains a codegenerator for emitting Golang source code
for datastructures based on IPLD Schemas.

It is reasonably complete.  See the [feature table](#completeness) below for details.

There is a CLI tool which can be used to run this generator.
See https://github.com/ipld/go-ipldtool !

Some features may still requiring writing code to fully configure them.
(PRs, here or in the go-ipldtool repo, welcome.)

See [README_behaviors](README_behaviors.md) for notes about the behaviors of the code output by the generator.

Check out the [HACKME](HACKME.md) document for more info about the internals,
how they're organized, and how to hack on this package.


aims
----

`gengo` aims to:

- generate native Golang code
- that faithfully represents the data structuring specified by an IPLD Schema,
- operating efficiently, both in speed (both creating and inspecting) and memory compactness;
- producing a better type system for Golang (we've got unions and enums!)
- that is both powerful and generic (when you need it)
- and minimalist (when you don't),
- with immutable data structures,
- good validation primitives and type-supported safety systems,
- and is friendly to embellishments of other hand-written Golang code.

Some of these aims should be satisfied.

Some are still a stretch ;)  (we definitely don't have "minimalist" outputs, yet.
Making this reachable by tuning is a goal, however!)


completeness
------------

Legend:

- `✔` - supported!
- `✘` - not currently supported.
- `⚠` - not currently supported -- and might not be obvious; be careful.
- `-` - is not applicable
- `?` - feature definition needed!  (applies to many of the "native extras" rows -- often there's partial features, but also room for more.)
- ` ` - table is not finished, please refer to the code and help fix the table :)

| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| structs                          |    ...    |    ...   |
| ... type level                   |     ✔     |     ✔    |
| ... native extras                |     ?     |     ?    |
| ... map representation           |     ✔     |     ✔    |
| ... ... including optional       |     ✔     |     ✔    |
| ... ... including renames        |     ✔     |     ✔    |
| ... ... including implicits      |     ⚠     |     ⚠    |
| ... tuple representation         |     ✔     |     ✔    |
| ... ... including optional       |     ✔     |     ✔    |
| ... ... including renames        |     -     |     -    |
| ... ... including implicits      |     ⚠     |     ⚠    |
| ... stringjoin representation    |     ✔     |     ✔    |
| ... ... including optional       |     -     |     -    |
| ... ... including renames        |     -     |     -    |
| ... ... including implicits      |     -     |     -    |
| ... stringpairs representation   |     ✘     |     ✘    |
| ... ... including optional       |           |          |
| ... ... including renames        |           |          |
| ... ... including implicits      |           |          |
| ... listpairs representation     |     ✘     |     ✘    |
| ... ... including optional       |           |          |
| ... ... including renames        |           |          |
| ... ... including implicits      |           |          |

| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| lists                            |    ...    |    ...   |
| ... type level                   |     ✔     |     ✔    |
| ... native extras                |     ?     |     ?    |
| ... list representation          |     ✔     |     ✔    |

| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| maps                             |    ...    |    ...   |
| ... type level                   |     ✔     |     ✔    |
| ... native extras                |     ?     |     ?    |
| ... map representation           |     ✔     |     ✔    |
| ... stringpairs representation   |     ✘     |     ✘    |
| ... listpairs representation     |     ✘     |     ✘    |

| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| unions                           |    ...    |    ...   |
| ... type level                   |     ✔     |     ✔    |
| ... keyed representation         |     ✔     |     ✔    |
| ... envelope representation      |     ✘     |     ✘    |
| ... kinded representation        |     ✔     |     ✔    |
| ... inline representation        |     ✘     |     ✘    |
| ... stringprefix representation  |     ✔     |     ✔    |
| ... byteprefix representation    |     ✘     |     ✘    |
 
| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| strings                          |     ✔     |     ✔    |
| bytes                            |     ✔     |     ✔    |
| ints                             |     ✔     |     ✔    |
| floats                           |     ✔     |     ✔    |
| bools                            |     ✔     |     ✔    |
| links                            |     ✔     |     ✔    |

| feature                          | accessors | builders |
|:---------------------------------|:---------:|:--------:|
| enums                            |    ...    |    ...   |
| ... type level                   |     ✘     |     ✘    |
| ... string representation        |     ✘     |     ✘    |
| ... int representation           |     ✘     |     ✘    |
