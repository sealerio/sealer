Testing Generated Code
======================

Getting nice testing scenarios for generated code is tricky.

There are three major phases of testing we can do:

1. unit tests on the codegen tooling itself
2. tests that the emitted generated code can compile
3. behavioral tests on the emitted generated code

Groups 1 and 2 are pretty easy (and we don't have a lot of group 1,
because group 2 covers it pretty nicely).

Group 3, however, is tricky.
The rest of the document will talk more about this kind of testing.


Behavioral Tests
----------------

### Behavioral tests are run via plugins

This package does some fancy footwork with the golang `plugin` system
and `go build -buildmode=plugin` in order to compile and load the
generated code into the same memory as the test process,
thereby letting us do behavioral tests on the gen'd code quite seamlessly.

This does have some downsides -- namely, the `plugin` system requires
the use of `cgo`.  This means you'll need more things installed on your
host system than just the go toolchain -- you might need `gcc` and friends too.
The `plugin` system also (at time of writing) doesn't support windows.
You can skip the behavioral tests if this is a problem: see the next section.

### Skipping the behavioral tests

You can skip behavioral tests by adding a build tag of `"skipgenbehavtests"`.
They'll also be automatically skipped if you're running in `"!cgo"` mode --
however, the `go` tools don't automatically set `"!cgo"` just because it
doesn't have enough tools, so you'll still need to be explicit about this.

The ability of the generated code to be compiled will still be checked,
even if the behavioral tests are skipped.

You can grep through your test output for "bhvtest=skip" to see at-a-glance
if the behavioral tests are being skipped.

### Plugins don't jive with race detection

Long story short: if running tests with the race detector, skip the gen tests.
Any `go test -race` is going to need `go test -race -tags 'skipgenbehavtests'`.

The go plugin system requires the plugin and host process have the same "runtime".
The way '-race' works makes for an effectively distinct runtime.

### Alternatives to plugins

Are there other ways this could be done?  Well, surely.  There always are.

Invoking 'go test' as a subprocess is usually central to alternative ideas,
but this has several downsides I haven't yet figured out how to counter:

- it tends to result in very difficult-to-wrangle output;
- it ends up imposing a lot of constraints on file organization,
  which in turn makes writing tests into a very high-friction endeavour;
- passing down flags to the test process (e.g., '-v' and friends)
  begins to require additional infrastructure;
- some flags such as '-run' are even yet more difficult to pass down usefully;
- every single behavioral test has to have an exported top-level function,
  making some things that are trivial with closures now difficult...
- You get the idea.

You can see some attempts around
commit 79de0e26469f0d2899c813a2c70d921fe5946f23 and its halfdozen or so
parents; remarks can be found in the git commit messages.

There are probably yet more variations in ways files and functions could
be reorganized, particularly to minimize the downsides of the file and
package splitting requirements, but if you're looking at this scenario and
wanting to propose one... Do read those commits to avoid getting into a
Chesterton's Fence situation, and kindly try it before proposing it.
Not all of the hurdles are immediately obvious.

### Plugins are only used in testing

This might not need a clarification statement, but just in case it does:
**plugins** (and by extention, cgo) **are not necessary**
for **doing** codegen nor for **using** the resulting generated code.
They are _only_ used for our testing of the codegen tooling
(and specifically, at that, for the behavioral tests).
We would not foist a dependency like cgo on codegen users.
