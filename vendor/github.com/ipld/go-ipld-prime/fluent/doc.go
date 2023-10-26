/*
The fluent package offers helper utilities for using NodeAssembler
more tersely by providing an interface that handles all errors for you,
and allows use of closures for any recursive assembly
so that creating trees of data results in indentation for legibility.

Note that the fluent package creates wrapper objects in order to provide
the API conveniences that it does, and this comes at some cost to performance.
If you're optimizing for performance, using some of the features of the
fluent package may be inadvisable (and some moreso than others).
However, as with any performance questions, benchmark before making decisions;
its entirely possible that your performance bottlenecks will be elsewhere
and there's no reason to deny yourself syntactic sugar if the costs don't
detectably affect the bottom line.
Various feature of the package will also attempt to document how costly they are in relative terms
(e.g. the fluent.Reflect helper methods are very costly;
*/
package fluent
