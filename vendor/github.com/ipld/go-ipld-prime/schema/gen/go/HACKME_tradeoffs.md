tradeoffs and design decisions in codegen
=========================================

In creating codegen for IPLD, as with any piece of software,
there are design decisions to be made, and tradeoffs to be considered.

Some of these decisions and tradeoffs are particularly interesting
in IPLD codegen because they're:

- significantly different resolutions and answers than the same decisions for non-codegen Node implementations
- able to make significantly different choices, expanding the decision space dimentionality, since they have more information before compile-time
- can reach higher upper bounds of performance, due to that pre-compile-time foothold
- have correspondingly less flexibility in many ways because of the same.


values we can balance
---------------------

Let's enumerate the things we can balance (and give them some short reference codes):

- AS: assembly/binary/final-shipping size, in bytes
- BM: builder memory, in bytes, used as long as a NodeBuilder is in use
- SP: execution speed, in seconds, especially of NodeBuilder in deserialization use
- AC: allocations, in count, needed for operations (though in truth, this is just a proxy for SP due to its outsized impact there)
- ERG: ergonomics, as an ineffable, ellusive-to-measurement sort of vibe of the thing, and how well it self-explains use and deters erroneous application
- GLOC: generated lines of code, as a line count or in bytes, of interest because it may be of noticable cost in version control weight

This list is in particular regarding concerns that come to light in considering performant deserialization operations...
however, it's fairly representative of general use as well:
traversals and serialization are generally easier situations to handle (they essentially get to skip the "BM" term);
and while different operations might encounter different scalars for how much these different values affect them,
as we'll see in the prioritization coming up in the next section... that turns out not to matter for our priorities.

We can also other code which knows it's addressing generated code can use special methods,
  which means we can in a way disregard its effect on this ordering (mostly).

Side note: though "AC" is *mostly* just a proxy for SP,
AC can also count on its own in *addition* to SP because it increases the *variance* in SP.
(But we don't often regard this; it's a pretty fine detail, and the goal is "minimize" either way.)


prioritization of those values
------------------------------

The designs here emerge from `SP > BM > AS`.

More fully: `SP|AC > BM > ERG > AS > GLOC`.

In other words: speed is the overwhelming priority;
thereafter, we'd like to conserve memory (but will readily sell it for speed);
ergonomics takes a side seat to both of these (the intention being that we can add 'porcelain' layers separately later);
assembly size is a concern but fourth fiddle (if this is your dominant concern, you may not want to use codegen, or may want a different library implementation that aims at the same specs);
and generated code size is a concern but we'll trade it away for any of the other priorities
(because it's a cost that doesn't end up affecting final users of products built with this system!).

(Some caveats: it's still possible to consider it a red flag if ratios on these get wild.
For example if BM gets > 2x, it's questionable;
and at some point we could imagine saying that AS has really gotten out of hand.)

(BM also has some special conditions such that if it increases on recursive kinds, but not on scalars,
we regard that as roughly half price, because generally most of a tree is leaves.
(As it happens, though, this has turned out not to change any results much.))

"Ergonomics" remains a tricky to account for.
It's true that when push comes to shove, speed and memory economy win.
But it's not at all single-dimentional; and with codegen, there are many options
which set a higher bar for all three concerns at the same time.
(In contrast, there's a stark upper limit to the ergonomic possbilities for
non-codegen no-schema handling of data -- code handling the data model has
the limits that its monomorphized approach imposes on it, and there's little
that can be done to avoid or improve upon that.)
