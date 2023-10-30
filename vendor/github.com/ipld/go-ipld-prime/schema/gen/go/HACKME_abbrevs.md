abbreviations
=============

A lot of abbreviations are used the generated code in the interest of minimizing the size of the output.

This is a short index of the most common ones:

- `n` -- **n**ode, of course -- the accessor functions on node implementations usually refer to their 'this' as 'n'.
- `na` -- **n**ode **a**ssembler
	- `la` or `ma` may also be seen for **l**ist and **m**ap in some contexts (but may refer to same type being called `na` in other contexts).
- `w` -- **w**ork-in-progress node -- you'll see this in nearly every assembler (e.g. `na.w` is a very common string).

inside nodes:

- `x` -- a placeholder for "the thing" for types that contain only one element of data (e.g., the string inside a codegen'd node of string kind).
- `t` -- **t**able -- the slice inside most map nodes that is used for alloc amortizations and maintaining order.
- `m` -- **m**ap -- the actual map inside most map nodes (seealso `t`, which is usually a sibling).

inside assemblers:

- `va` -- **v**alue **a**ssembler -- an assembler for values in lists or maps (often embedded in the node asembler, e.g. `ma.va` and `la.va` are common strings).
- `ka` -- **k**ey **a**ssembler -- an assembler for keys in maps (often embedded in the node asembler, e.g. `ma.ka` is a common string).
- `ca_*` -- **c**hild **a**ssembler -- the same concept as `ka` and `va`, but appearing in structs and other types that have differentiated children.
- `cm` -- **c**hild **m**aybe -- a piece of memory sometimes found in a node assembler for statekeeping for child assemblers.
- `m` -- **m**aybe pointer -- a pointer to where an assembler should put a mark when it's finished.  (this is often a pointer to a parent structure's 'cm'!)
