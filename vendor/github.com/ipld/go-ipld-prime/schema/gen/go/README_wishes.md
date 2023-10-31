wishes and dreams of other things that could be
===============================================

### minimal gen for regular code

(In other short words: gen without the monomorphization.)

It'd be neat to have a mode that generates all public fields (shrugging on immutability),
and can also toggle off generating all the data model Node interface satisfaction methods.
This would let us use IPLD Schemas to define types (including enums and unions) and get Golang code out,
and easily use it in programs even where we don't particularly care about the Node interface.
(E.g., suppose we want an enum or a union type -- which Go doesn't naturally have -- but aren't going to use it
for serialization or anything else.  We've already got a lot of codegen here; why not make it help there too?)

It's unclear if we can load that much into this gen project, or if it'd be easier to make another one for this.
The output code would definitely be substantially structurally different;
and it would also tend to be useless/silly to generate different parts of a type system in each of the different modes,
so it's pretty likely that any practical user story would involve different process invocations to use them anyway.
