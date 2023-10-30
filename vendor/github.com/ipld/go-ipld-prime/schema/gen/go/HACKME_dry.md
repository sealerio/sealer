HACKME: "don't repeat yourself": how-to (and, limits of that goal)
==================================================================

Which kind of extraction applies?
---------------------------------

Things vary in how identical they are.

- **Textually identical**: Some code is textually identical between different types,
  varying only in the most simple and obvious details, like the actual type name it's attached to.
	- These cases can often be extracted on the generation side...
		- We tend to put them in `genparts{*}.go` files.
	- But the output is still pretty duplicacious.

- **Textually identical minus simple variations**: Some code is textually *nearly* identical,
  but varies in relatively minor ways (such as whether or not the "Repr" is part of munges, and "Representation()" calls are made, etc).
	- These cases can often be extracted on the generation side...
		- We tend to put them in `genparts{*}.go` files.
		- There's just a bit more `{{ various templating }}` injected in them, compared to other textually identical templates.
	- But the output is still pretty duplicacious.

- **Linkologically identical**: When code is not _only_ textually identical,
  but also refers to identical types.
	- These cases can be extracted on the generation side...
		- but it may be questionable to do so: if its terse enough in the output, there's that much less incentive to make a template-side shorthand for it.
	- The output in this case can actually be deduplicated!
		- It's possible we haven't bothered yet.  **That doesn't mean it's not worth it**; we probably just haven't had time yet.  PRs welcome.
		- How?
			- functions?  This is the most likely to apply.
			- embedded types?  We haven't seen many cases where this can help, yet (unfortunately?).
			- shared constants?
		- It's not always easy to do this.
			- We usually put something in the "minima" file.
			- We don't currently have a way to toggle whether whole features or shared constants are emitted in the minima file.  Todo?
				- This requires keeping state that records what's necessary as we go, so that we can do them all together at the end.
				- May also require varying the imports at the top of the minima file.  (But: by doing it only here, we can avoid that complexity in every other file.)
	- **This is actually pretty rare**.  _Things that are textually identical are not necessarily linkologically identical_.
		- One can generally turn things that are textually identical into linkologically identical by injecting an interface into the types...
			- ... but this isn't always a *good* idea:
				- if this would cause more allocations?  Yeah, very no.
				- even if this can be done without a heap allocation, it probably means inlining and other optimizations will become impossible for the compiler to perform, and often, we're not okay with the performance implications of that either.

- **Identical if it wasn't for debugability**: In some cases, code varies only by some small constants...
  and really, those constants could be removed entirely.  If... we didn't care about debugging.  Which we do.
	- This is really the same as "textually identical minus simple variations", but worth talking about briefly just because of the user story around it.
	- A bunch of the error-thunking methods on Node and NodeAssemblers exemplify this.
		- It's really annoying that we can't remove this boilerplate entirely from the generated code.
		- It's also basically impossible, because we *want* information that varies per type in those error messages.


What mechanism of extraction should be used?
--------------------------------------------

- (for gen-side dry) gen-side functions
	- this is most of what we've done so far
- (for gen-side dry) sub-templates
	- we currently don't really use this at all
- (for gen-side dry) template concatenation
	- some of this: kinded union representations do this
- (for output-side dry) output side functions
	- some of this: see "minima" file.
- (for output-side dry) output side embeds
	- we currently don't really use this at all (it hasn't really turned out applicable in any cases yet).


Don't overdo it
---------------

I'd rather have longer templates than harder-to-read and harder-to-maintain templates.

There's a balance to this and it's tricky to pick out.

A good heuristic to consider might be: are we extracting this thing because we can?
Or because if we made changes to this thing in the future, we'd expect to need to make that change in every single place we've extracted it from,
which therefore makes the extraction a net win for maintainability?
If it's the latter: then yes, extract it.
If it's not clear: maybe let it be.

(It may be the case that the preferable balance for DRYing changes over time as we keep maintaining things.
We'll see; but it's certainly the case that the first draft of this package has favored length heavily.
There was a lot of "it's not clear" when the maintainability heuristic was applied during the first writing of this;
that may change!  If so, that's great.)
