# go-graphsync changelog

# go-graphsync v0.11.0

Breaking update to new go-datastore interfaces

### Changelog

- github.com/ipfs/go-graphsync:
  - Merge branch 'release/v0.10.6'
  - update to context datastores (#275) ([ipfs/go-graphsync#275](https://github.com/ipfs/go-graphsync/pull/275))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Whyrusleeping | 1 | +895/-111 | 3 |

# go-graphsync v0.10.6

Use TaskQueue in ResponseManager and remove memory backpressure from request side

### Changelog

- github.com/ipfs/go-graphsync:
  - feat!(requestmanager): remove request allocation backpressure (#272) ([ipfs/go-graphsync#272](https://github.com/ipfs/go-graphsync/pull/272))
  - message/pb: stop using gogo/protobuf (#277) ([ipfs/go-graphsync#277](https://github.com/ipfs/go-graphsync/pull/277))
  - mark all test helper funcs via t.Helper (#276) ([ipfs/go-graphsync#276](https://github.com/ipfs/go-graphsync/pull/276))
  - chore(queryexecutor): remove unused RunTraversal
  - chore(responsemanager): remove unused workSignal
  - chore(queryexecutor): fix tests for runtraversal refactor + clean up
  - feat(queryexecutor): merge RunTraversal into QueryExecutor
  - feat(responsemanager): QueryExecutor to separate module - use TaskQueue, add tests
  - Merge branch 'release/v0.10.5'

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Rod Vagg | 5 | +1451/-1213 | 28 |
| hannahhoward | 1 | +150/-120 | 9 |
| Daniel Martí | 2 | +133/-122 | 19 |

# go-graphsync v0.10.5

Small refactors and improvements, remove memory leaks, add OutgoingRequestProcessing hook

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(responseassembler): dont hold block data reference in passed on subscribed block link (#268) ([ipfs/go-graphsync#268](https://github.com/ipfs/go-graphsync/pull/268))
  - sync: update CI config files (#266) ([ipfs/go-graphsync#266](https://github.com/ipfs/go-graphsync/pull/266))
  - Check IPLD context cancellation error type instead of string comparison
  - Use `context.CancelFunc` instead of `func()` (#257) ([ipfs/go-graphsync#257](https://github.com/ipfs/go-graphsync/pull/257))
  - fix: bail properly when budget exceeded
  - feat(requestmanager): report inProgressRequestCount on OutgoingRequests event
  - fix(requestmanager): remove failing racy test select block
  - feat(requestmanager): add OutgoingRequeustProcessingListener
  - Merge branch 'release/v0.10.4'

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Rod Vagg | 4 | +205/-91 | 13 |
| Masih H. Derkani | 2 | +49/-24 | 9 |
| Hannah Howard | 1 | +30/-11 | 1 |
| web3-bot | 1 | +39/-0 | 4 |

# go-grapshync 0.10.4

Fix a critical bug in the allocator

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(allocator): prevent buffer overflow (#248) ([ipfs/go-graphsync#248](https://github.com/ipfs/go-graphsync/pull/248))
  - Merge branch 'release/v0.10.3'

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +43/-3 | 2 |

# go-graphsync 0.10.3

Additional config options and metrics

### Changelog

- github.com/ipfs/go-graphsync:
  - Configure message parameters (#247) ([ipfs/go-graphsync#247](https://github.com/ipfs/go-graphsync/pull/247))
  - Stats! (#246) ([ipfs/go-graphsync#246](https://github.com/ipfs/go-graphsync/pull/246))
  - Limit simultaneous incoming requests on a per peer basis (#245) ([ipfs/go-graphsync#245](https://github.com/ipfs/go-graphsync/pull/245))
  - sync: update CI config files (#191) ([ipfs/go-graphsync#191](https://github.com/ipfs/go-graphsync/pull/191))
  - Merge branch 'release/v0.10.2'

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 3 | +261/-67 | 14 |
| web3-bot | 1 | +214/-82 | 11 |

# go-graphsync 0.10.2

Fix minor deadlocking issue in notification system

### Changelog

- github.com/ipfs/go-graphsync:
  - test(responsemanager): fix flakiness TestCancellationViaCommand (#243) ([ipfs/go-graphsync#243](https://github.com/ipfs/go-graphsync/pull/243))
  - Fix deadlock on notifications (#242) ([ipfs/go-graphsync#242](https://github.com/ipfs/go-graphsync/pull/242))
  - Merge branch 'release/v0.10.1'

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 2 | +66/-25 | 5 |

# go-graphsync 0.10.1

Minor fix to allocation behavior on request side

### Changelog
- github.com/ipfs/go-graphsync:
  - Free memory on request finish (#240) ([ipfs/go-graphsync#240](https://github.com/ipfs/go-graphsync/pull/240))
  - release: v1.10.0 ([ipfs/go-graphsync#238](https://github.com/ipfs/go-graphsync/pull/238))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +36/-21 | 9 |

# go-graphsync 0.10.0

- github.com/ipfs/go-graphsync:
  - feat: update to go-ipld-prime v0.12.3 (#237) ([ipfs/go-graphsync#237](https://github.com/ipfs/go-graphsync/pull/237))
  - Add support for IPLD prime's budgets feature in selectors (#235) ([ipfs/go-graphsync#235](https://github.com/ipfs/go-graphsync/pull/235))
  - feat(graphsync): add an index for blocks in the on new block hook (#234) ([ipfs/go-graphsync#234](https://github.com/ipfs/go-graphsync/pull/234))
  - Do not send first blocks extension (#230) ([ipfs/go-graphsync#230](https://github.com/ipfs/go-graphsync/pull/230))
  - Protect Libp2p Connections (#229) ([ipfs/go-graphsync#229](https://github.com/ipfs/go-graphsync/pull/229))
  - test(responsemanager): remove check (#228) ([ipfs/go-graphsync#228](https://github.com/ipfs/go-graphsync/pull/228))
  - feat(graphsync): give missing blocks a named error (#227) ([ipfs/go-graphsync#227](https://github.com/ipfs/go-graphsync/pull/227))
  - Add request limits (#224) ([ipfs/go-graphsync#224](https://github.com/ipfs/go-graphsync/pull/224))
  - Tech Debt Cleanup and Docs Update (#219) ([ipfs/go-graphsync#219](https://github.com/ipfs/go-graphsync/pull/219))

Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 8 | +2988/-2398 | 79 |
| dirkmc | 1 | +3/-3 | 2 |

# go-graphsync 0.9.3

Hotfix for 0.9.2

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(impl): use correct allocator

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 1 | +1/-1 | 1 |

# go-graphsync 0.9.2

DO NOT USE: Contains bug

Minor bug fix and thread unblock

### Changelog
- github.com/ipfs/go-graphsync:
  - fix(requestmanager): remove main thread block on allocation (#216) ([ipfs/go-graphsync#216](https://github.com/ipfs/go-graphsync/pull/216))
  - feat(allocator): add debug logging (#213) ([ipfs/go-graphsync#213](https://github.com/ipfs/go-graphsync/pull/213))
  - fix: spurious warn log (#210) ([ipfs/go-graphsync#210](https://github.com/ipfs/go-graphsync/pull/210))
  - docs(CHANGELOG): update for v0.9.1 release (#212) ([ipfs/go-graphsync#212](https://github.com/ipfs/go-graphsync/pull/212))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 3 | +52/-31 | 7 |
| dirkmc | 1 | +3/-1 | 1 |


# go-graphsync 0.9.1

Fix a critical bug in the message builder

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(message): fix dropping of response extensions (#211) ([ipfs/go-graphsync#211](https://github.com/ipfs/go-graphsync/pull/211))
  - docs(CHANGELOG): update change log ([ipfs/go-graphsync#208](https://github.com/ipfs/go-graphsync/pull/208))
  - docs(README): add notice about branch rename

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +60/-0 | 2 |
| hannahhoward | 2 | +39/-2 | 3 |

# go-graphsync 0.9.0

This release unifies the master branch with the 0.6.x branch, which contained several divergent features

### Changelog

- github.com/ipfs/go-graphsync:
  - feat(deps): update go-ipld-prime v0.12.0 (#206) ([ipfs/go-graphsync#206](https://github.com/ipfs/go-graphsync/pull/206))
  - fix(graphsync): make sure linkcontext is passed (#207) ([ipfs/go-graphsync#207](https://github.com/ipfs/go-graphsync/pull/207))
  - Merge final v0.6.x commit history, and 0.8.0 changelog (#205) ([ipfs/go-graphsync#205](https://github.com/ipfs/go-graphsync/pull/205))
  - Fix broken link to IPLD selector documentation (#189) ([ipfs/go-graphsync#189](https://github.com/ipfs/go-graphsync/pull/189))
  - fix: check errors before defering a close (#200) ([ipfs/go-graphsync#200](https://github.com/ipfs/go-graphsync/pull/200))
  - chore: fix checks (#197) ([ipfs/go-graphsync#197](https://github.com/ipfs/go-graphsync/pull/197))
  - Merge the v0.6.x commit history (#190) ([ipfs/go-graphsync#190](https://github.com/ipfs/go-graphsync/pull/190))
  - Ready for universal CI (#187) ([ipfs/go-graphsync#187](https://github.com/ipfs/go-graphsync/pull/187))
  - fix(requestmanager): pass through linksystem (#166) ([ipfs/go-graphsync#166](https://github.com/ipfs/go-graphsync/pull/166))
  - fix missing word in section title (#179) ([ipfs/go-graphsync#179](https://github.com/ipfs/go-graphsync/pull/179))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 10 | +2452/-1125 | 110 |
| Aarsh Shah | 2 | +40/-177 | 6 |
| dirkmc | 4 | +118/-11 | 8 |
| hannahhoward | 1 | +81/-11 | 6 |
| aarshkshah1992 | 3 | +87/-3 | 7 |
| Steven Allen | 2 | +20/-53 | 4 |
| Dirk McCormick | 1 | +11/-0 | 1 |
| Masih H. Derkani | 1 | +1/-1 | 1 |
| Ismail Khoffi | 1 | +1/-1 | 1 |

# go-graphsync 0.8.0

This release updates to the v0.9.0 branch of go-ipld-prime and adds a "trusted store" optimization that may produce important speed improvements.

It also includes several improvements to the internal testplan & updated
architecture docs.

### Changelog

- github.com/ipfs/go-graphsync:
  - Update for LinkSystem (#161) ([ipfs/go-graphsync#161](https://github.com/ipfs/go-graphsync/pull/161))
  - Round out diagnostic parameters (#157) ([ipfs/go-graphsync#157](https://github.com/ipfs/go-graphsync/pull/157))
  - map response codes to names (#148) ([ipfs/go-graphsync#148](https://github.com/ipfs/go-graphsync/pull/148))
  - Discard http output (#156) ([ipfs/go-graphsync#156](https://github.com/ipfs/go-graphsync/pull/156))
  - Add debug logging (#121) ([ipfs/go-graphsync#121](https://github.com/ipfs/go-graphsync/pull/121))
  - Add optional HTTP comparison (#153) ([ipfs/go-graphsync#153](https://github.com/ipfs/go-graphsync/pull/153))
  - docs(architecture): update architecture docs (#154) ([ipfs/go-graphsync#154](https://github.com/ipfs/go-graphsync/pull/154))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 5 | +885/-598 | 55 |
| dirkmc | 1 | +79/-50 | 2 |
| Aarsh Shah | 1 | +2/-6 | 2 |

# go-graphsync 0.7.0

This is a small release to update some dependencies. Importantly, it pulls in go-ipld-prime with
some significant breaking changes.

### Changelog

- github.com/ipfs/go-graphsync:
  - chore: update deps (#151) ([ipfs/go-graphsync#151](https://github.com/ipfs/go-graphsync/pull/151))
  - Automatically record heap profiles in testplans (#147) ([ipfs/go-graphsync#147](https://github.com/ipfs/go-graphsync/pull/147))
  - feat(deps): update go-ipld-prime v0.7.0 (#145) ([ipfs/go-graphsync#145](https://github.com/ipfs/go-graphsync/pull/145))
  - Release/v0.6.0 ([ipfs/go-graphsync#144](https://github.com/ipfs/go-graphsync/pull/144))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 2 | +3316/-3015 | 25 |
| Steven Allen | 1 | +95/-227 | 5 |

# go-graphsync 0.6.9

This release adds additional log statements and addresses a memory performance bug on the requesting side when making lots of outgoing requests at once

### Changelog

- github.com/ipfs/go-graphsync:
  - Back pressure incoming responses ([ipfs/go-graphsync#204](https://github.com/ipfs/go-graphsync/pull/204))
  - Log unverified blockstore memory consumption ([ipfs/go-graphsync#201](https://github.com/ipfs/go-graphsync/pull/201))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 5 | +1535/-381 | 25 |
| Aarsh Shah | 5 | +27/-17 | 5 |

# go-graphsync 0.6.8

### Changelog

- github.com/ipfs/go-graphsync:
  - refactor: replace particular request not found errors with public error (#188) ([ipfs/go-graphsync#188](https://github.com/ipfs/go-graphsync/pull/188))
  - fix(responsemanager): fix error codes (#182) ([ipfs/go-graphsync#182](https://github.com/ipfs/go-graphsync/pull/182))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +100/-51 | 5 |
| dirkmc | 1 | +10/-3 | 2 |

# go-graphsync 0.6.7

### Changelog

- github.com/ipfs/go-graphsync:
  - Add cancel request and wait function (#185) ([ipfs/go-graphsync#185](https://github.com/ipfs/go-graphsync/pull/185))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +154/-32 | 9 |
# go-graphsync 0.6.6

### Changelog

- github.com/ipfs/go-graphsync:
  - feat(requestmanager): add request timing (#181) ([ipfs/go-graphsync#181](https://github.com/ipfs/go-graphsync/pull/181))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +9/-1 | 1 |

# go-graphsync 0.6.5

### Changelog

- github.com/ipfs/go-graphsync:
  - Resolve 175 race condition, no change to hook timing (#178) ([ipfs/go-graphsync#178](https://github.com/ipfs/go-graphsync/pull/178))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +199/-171 | 10 |

# go-graphsync 0.6.4

### Changelog

- github.com/ipfs/go-graphsync:
  - feat/request-queued-hook (#172) ([ipfs/go-graphsync#172](https://github.com/ipfs/go-graphsync/pull/172))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| aarshkshah1992 | 3 | +87/-3 | 7 |
| dirkmc | 1 | +11/-0 | 1 |

# go-graphsync 0.6.3

### Changelog

- github.com/ipfs/go-graphsync:
  - Fix/log blockstore reads (#169) ([ipfs/go-graphsync#169](https://github.com/ipfs/go-graphsync/pull/169))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Aarsh Shah | 2 | +40/-177 | 6 |

# go-graphsync 0.6.2

### Changelog

- github.com/ipfs/go-graphsync:
  - Better logging for Graphsync traversal (#167) ([ipfs/go-graphsync#167](https://github.com/ipfs/go-graphsync/pull/167))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Aarsh Shah | 1 | +18/-2 | 2 |

# go-graphsync 0.6.1

### Changelog

- github.com/ipfs/go-graphsync:
  - feat: fire network error when network disconnects during request (#164) ([ipfs/go-graphsync#164](https://github.com/ipfs/go-graphsync/pull/164))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| dirkmc | 1 | +86/-8 | 4 |


# go-graphsync 0.6.0

Major code refactor for simplicity, ease of understanding

### Changelog

- github.com/ipfs/go-graphsync:
  - Merge branch 'master' into release/v0.6.0
  - move block allocation into message queue (#140) ([ipfs/go-graphsync#140](https://github.com/ipfs/go-graphsync/pull/140))
  - Response Assembler Refactor (#138) ([ipfs/go-graphsync#138](https://github.com/ipfs/go-graphsync/pull/138))
  - Add error listener on receiver (#136) ([ipfs/go-graphsync#136](https://github.com/ipfs/go-graphsync/pull/136))
  - Run testplan on in CI (#137) ([ipfs/go-graphsync#137](https://github.com/ipfs/go-graphsync/pull/137))
  - fix(responsemanager): fix network error propogation (#133) ([ipfs/go-graphsync#133](https://github.com/ipfs/go-graphsync/pull/133))
  - testground test for graphsync (#132) ([ipfs/go-graphsync#132](https://github.com/ipfs/go-graphsync/pull/132))
  - docs(CHANGELOG): update for v0.5.2 ([ipfs/go-graphsync#130](https://github.com/ipfs/go-graphsync/pull/1

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Alex Cruikshank | 4 | +3269/-1919 | 47 |
| Hannah Howard | 3 | +777/-511 | 25 |
| hannahhoward | 1 | +34/-13 | 3 |

# go-graphsync 0.5.2

Minor release resolves bugs in notification system

### Changelog

- github.com/ipfs/go-graphsync:
  - RegisterNetworkErrorListener should fire when there's an error connecting to the peer (#127) ([ipfs/go-graphsync#127](https://github.com/ipfs/go-graphsync/pull/127))
  - Permit multiple data subscriptions per original topic (#128) ([ipfs/go-graphsync#128](https://github.com/ipfs/go-graphsync/pull/128))
  - release: v0.5.1 (#123) ([ipfs/go-graphsync#123](https://github.com/ipfs/go-graphsync/pull/123))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| dirkmc | 2 | +272/-185 | 10 |
| Alex Cruikshank | 1 | +188/-110 | 12 |
| Hannah Howard | 1 | +23/-6 | 3 |

# go-graphsync 0.5.1

### Changelog

- github.com/ipfs/go-graphsync:
  - feat(responsemanager): allow configuration of max requests (#122) ([ipfs/go-graphsync#122](https://github.com/ipfs/go-graphsync/pull/122))

Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +23/-6 | 3 |

# go-graphsync 0.4.3

Update libp2p to 0.12. This libp2p release includes a breaking change to the libp2p stream interfaces.

### Changelog

- github.com/ipfs/go-graphsync:
  - feat: use go-libp2p-core 0.7.0 stream interfaces (#116) ([ipfs/go-graphsync#116](https://github.com/ipfs/go-graphsync/pull/116))

Contributors

| Contributor  | Commits | Lines ±  | Files Changed |
|--------------|---------|----------|---------------|
| Steven Allen |       1 | +195/-24 |             3 |

# go-graphsync 0.4.3

Minor fixes and patches

### Changelog

- github.com/ipfs/go-graphsync:
  - chore(benchmarks): remove extra files
  - fix(peerresponsemanager): avoid race condition that could result in NPE in link tracker (#118) ([ipfs/go-graphsync#118](https://github.com/ipfs/go-graphsync/pull/118))
  - docs(CHANGELOG): update for 0.4.2 ([ipfs/go-graphsync#117](https://github.com/ipfs/go-graphsync/pull/117))
  - feat(memory): improve memory usage (#110) ([ipfs/go-graphsync#110](https://github.com/ipfs/go-graphsync/pull/110))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 2 | +49/-6 | 7 |
| hannahhoward | 1 | +0/-0 | 2 |

# go-graphsync 0.4.2

bug fix for 0.4.1

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(notifications): fix lock in close (#115) ([ipfs/go-graphsync#115](https://github.com/ipfs/go-graphsync/pull/115))
  - docs(CHANGELOG): update for v0.4.1 ([ipfs/go-graphsync#114](https://github.com/ipfs/go-graphsync/pull/114))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +7/-0 | 1 |

# go-graphsync 0.4.1

critical bug fix for 0.4.0

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(allocator): remove peer from peer status list
  - docs(CHANGELOG): update for v0.4.0

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 2 | +23/-2 | 3 |

# go-graphsync 0.4.0

Feature release - add memory backpressure to responses to minimize extra memory usage

### Changelog

- github.com/ipfs/go-graphsync:
  - docs(CHANGELOG): update for 0.3.1 ([ipfs/go-graphsync#112](https://github.com/ipfs/go-graphsync/pull/112))
  - Update ipld-prime (#111) ([ipfs/go-graphsync#111](https://github.com/ipfs/go-graphsync/pull/111))
  - Add allocator for memory backpressure (#108) ([ipfs/go-graphsync#108](https://github.com/ipfs/go-graphsync/pull/108))
  - Shutdown notifications go routines (#109) ([ipfs/go-graphsync#109](https://github.com/ipfs/go-graphsync/pull/109))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 3 | +724/-83 | 18 |


# go-graphsync 0.3.1

Security fix -- switch to google protobufs

### Changelog

- github.com/ipfs/go-graphsync:
  - Switch to google protobuf generator (#105) ([ipfs/go-graphsync#105](https://github.com/ipfs/go-graphsync/pull/105))
  - feat(CHANGELOG): update for 0.3.0 ([ipfs/go-graphsync#104](https://github.com/ipfs/go-graphsync/pull/104))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +472/-1553 | 8 |

# go-graphsync 0.3.0

Significant updates allow for:
- completed response hooks run when response is done going over wire (or at least transmitted)
- listening for when blocks are actually sent
- being notified of network send errors on responder

### Changelog

- github.com/ipfs/go-graphsync:
  - docs(CHANGELOG): update for 0.2.1 ([ipfs/go-graphsync#103](https://github.com/ipfs/go-graphsync/pull/103))
  - Track actual network operations in a response (#102) ([ipfs/go-graphsync#102](https://github.com/ipfs/go-graphsync/pull/102))
  - feat(responsecache): prune blocks more intelligently (#101) ([ipfs/go-graphsync#101](https://github.com/ipfs/go-graphsync/pull/101))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 2 | +1983/-927 | 29 |

# go-graphsync 0.2.1

Compatibility fix for 0.2.0

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(metadata): fix cbor-gen (#98) ([ipfs/go-graphsync#98](https://github.com/ipfs/go-graphsync/pull/98))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +12/-16 | 3 |

# go-graphsync 0.2.0

Update to IPLD prime + several optimizations for performance

### Changelog

- github.com/ipfs/go-graphsync:
  - style(imports): fix imports
  - fix(selectorvalidator): memory optimization (#97) ([ipfs/go-graphsync#97](https://github.com/ipfs/go-graphsync/pull/97))
  - Update go-ipld-prime@v0.5.0 (#92) ([ipfs/go-graphsync#92](https://github.com/ipfs/go-graphsync/pull/92))
  - refactor(metadata): use cbor-gen encoding (#96) ([ipfs/go-graphsync#96](https://github.com/ipfs/go-graphsync/pull/96))
  - Release/v0.1.2 ([ipfs/go-graphsync#95](https://github.com/ipfs/go-graphsync/pull/95))
  - Return Request context cancelled error (#93) ([ipfs/go-graphsync#93](https://github.com/ipfs/go-graphsync/pull/93))
  - feat(benchmarks): add p2p stress test (#91) ([ipfs/go-graphsync#91](https://github.com/ipfs/go-graphsync/pull/91))
- github.com/hannahhoward/cbor-gen-for (null -> v0.0.0-20200817222906-ea96cece81f1):
  - add flag to select map encoding ([hannahhoward/cbor-gen-for#1](https://github.com/hannahhoward/cbor-gen-for/pull/1))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Eric Myhre | 1 | +2919/-121 | 39 |
| Hannah Howard | 3 | +412/-103 | 15 |
| hannahhoward | 1 | +31/-31 | 7 |
| whyrusleeping | 1 | +31/-18 | 2 |
| Aarsh Shah | 1 | +27/-1 | 3 |

# go-graphsync 0.1.2

Minor release with initial benchmarks

### Changelog

- github.com/ipfs/go-graphsync:
  - Benchmark framework + First memory fixes (#89) ([ipfs/go-graphsync#89](https://github.com/ipfs/go-graphsync/pull/89))
  - docs(CHANGELOG): update for v0.1.1 ([ipfs/go-graphsync#85](https://github.com/ipfs/go-graphsync/pull/85))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +1055/-39 | 17 |

# go-graphsync 0.1.1

Minor fix for alternate persistence stores and deduplication

### Changelog

- github.com/ipfs/go-graphsync:
  - docs(CHANGELOG): update for v0.1.0 release ([ipfs/go-graphsync#84](https://github.com/ipfs/go-graphsync/pull/84))
  - Dedup by key extension (#83) ([ipfs/go-graphsync#83](https://github.com/ipfs/go-graphsync/pull/83))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 1 | +316/-7 | 10 |

# go-graphsync v0.1.0

Major release (we fell behind on creating tagged releases for a while) -- many augmentations to hooks, authorization, persistence, request execution.

### Changelog

- github.com/ipfs/go-graphsync:
  - style(imports): fix import formatting
  - feat(persistenceoptions): add unregister ability (#80) ([ipfs/go-graphsync#80](https://github.com/ipfs/go-graphsync/pull/80))
  - fix(message): regen protobuf code (#79) ([ipfs/go-graphsync#79](https://github.com/ipfs/go-graphsync/pull/79))
  - feat(requestmanager): run response hooks on completed requests (#77) ([ipfs/go-graphsync#77](https://github.com/ipfs/go-graphsync/pull/77))
  - Revert "add extensions on complete (#76)"
  - add extensions on complete (#76) ([ipfs/go-graphsync#76](https://github.com/ipfs/go-graphsync/pull/76))
  - All changes to date including pause requests & start paused, along with new adds for cleanups and checking of execution (#75) ([ipfs/go-graphsync#75](https://github.com/ipfs/go-graphsync/pull/75))
  - More fine grained response controls (#71) ([ipfs/go-graphsync#71](https://github.com/ipfs/go-graphsync/pull/71))
  - Refactor request execution and use IPLD SkipMe functionality for proper partial results on a request (#70) ([ipfs/go-graphsync#70](https://github.com/ipfs/go-graphsync/pull/70))
  - feat(graphsync): implement do-no-send-cids extension (#69) ([ipfs/go-graphsync#69](https://github.com/ipfs/go-graphsync/pull/69))
  - Incoming Block Hooks (#68) ([ipfs/go-graphsync#68](https://github.com/ipfs/go-graphsync/pull/68))
  - fix(responsemanager): add nil check (#67) ([ipfs/go-graphsync#67](https://github.com/ipfs/go-graphsync/pull/67))
  - Add autocomment configuration
  - refactor(hooks): use external pubsub (#65) ([ipfs/go-graphsync#65](https://github.com/ipfs/go-graphsync/pull/65))
  - Update of IPLD Prime (#66) ([ipfs/go-graphsync#66](https://github.com/ipfs/go-graphsync/pull/66))
  - Add standard issue template
  - feat(responsemanager): add listener for completed responses (#64) ([ipfs/go-graphsync#64](https://github.com/ipfs/go-graphsync/pull/64))
  - Update Requests (#63) ([ipfs/go-graphsync#63](https://github.com/ipfs/go-graphsync/pull/63))
  - Add pausing and unpausing of requests (#62) ([ipfs/go-graphsync#62](https://github.com/ipfs/go-graphsync/pull/62))
  - ci(circle): remove benchmark task for now
  - ci(circle): update orb
  - Outgoing Request Hooks, swapping persistence layers (#61) ([ipfs/go-graphsync#61](https://github.com/ipfs/go-graphsync/pull/61))
  - Feat/request hook loader chooser (#60) ([ipfs/go-graphsync#60](https://github.com/ipfs/go-graphsync/pull/60))
  - Option to Reject requests by default (#58) ([ipfs/go-graphsync#58](https://github.com/ipfs/go-graphsync/pull/58))
  - Testify refactor (#56) ([ipfs/go-graphsync#56](https://github.com/ipfs/go-graphsync/pull/56))
  - Switch To Circle CI (#57) ([ipfs/go-graphsync#57](https://github.com/ipfs/go-graphsync/pull/57))
  - fix(deps): go mod tidy
  - docs(README): remove ipldbridge reference
  - Tech Debt: Remove IPLD Bridge ([ipfs/go-graphsync#55](https://github.com/ipfs/go-graphsync/pull/55))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Hannah Howard | 20 | +13273/-7718 | 262 |
| hannahhoward | 13 | +1663/-1906 | 184 |
| Hector Sanjuan | 2 | +95/-0 | 3 |

# go-graphsync v0.0.5

Minor release -- update task queue and add some documentation

### Changelog

- github.com/ipfs/go-graphsync:
  - feat: update the peer task queue ([ipfs/go-graphsync#54](https://github.com/ipfs/go-graphsync/pull/54))
  - docs(readme): document the storeutil package in the readme ([ipfs/go-graphsync#52](https://github.com/ipfs/go-graphsync/pull/52))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| Steven Allen | 2 | +68/-49 | 5 |

# go-graphsync 0.0.4

Initial release to incorporate into go-data-transfer module.

Implements request authorization, request hooks, default valdiation policy, etc

### Changelog

- github.com/ipfs/go-graphsync:
  - Add DAG Protobuf Support ([ipfs/go-graphsync#51](https://github.com/ipfs/go-graphsync/pull/51))
  - Add response hooks ([ipfs/go-graphsync#50](https://github.com/ipfs/go-graphsync/pull/50))
  - Request hooks ([ipfs/go-graphsync#49](https://github.com/ipfs/go-graphsync/pull/49))
  - Add a default validation policy ([ipfs/go-graphsync#48](https://github.com/ipfs/go-graphsync/pull/48))
  - Send user extensions in request ([ipfs/go-graphsync#47](https://github.com/ipfs/go-graphsync/pull/47))
  - Revert "Merge pull request #44 from ipfs/chore/update-peertaskqueue"
  - Update peertaskqueue ([ipfs/go-graphsync#44](https://github.com/ipfs/go-graphsync/pull/44))
  - Refactor file organization ([ipfs/go-graphsync#43](https://github.com/ipfs/go-graphsync/pull/43))
  - feat(graphsync): support extension protocol ([ipfs/go-graphsync#42](https://github.com/ipfs/go-graphsync/pull/42))
  - Bump go-ipld-prime to 092ea9a7696d ([ipfs/go-graphsync#41](https://github.com/ipfs/go-graphsync/pull/41))
  - Fix some typo ([ipfs/go-graphsync#40](https://github.com/ipfs/go-graphsync/pull/40))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 12 | +3040/-1516 | 103 |
| Hannah Howard | 2 | +253/-321 | 3 |
| Dirk McCormick | 1 | +47/-33 | 4 |
| Edgar Lee | 1 | +36/-20 | 8 |
| Alexey | 1 | +15/-15 | 1 |

# go-graphsync 0.0.3

Bug fix release. Fix issues issues with message queue.

### Changelog

- github.com/ipfs/go-graphsync:
  - fix(messagequeue): no retry after queue shutdown ([ipfs/go-graphsync#38](https://github.com/ipfs/go-graphsync/pull/38))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 1 | +70/-1 | 2 |

# go-graphsync 0.0.2

Bug fix release. Fix message sizes to not overflow limits.

### Changelog

- github.com/ipfs/go-graphsync:
  - Limit Response Size ([ipfs/go-graphsync#37](https://github.com/ipfs/go-graphsync/pull/37))

### Contributors

| Contributor | Commits | Lines ± | Files Changed |
|-------------|---------|---------|---------------|
| hannahhoward | 2 | +295/-52 | 5 |

# go-graphysnc 0.0.1-filecoin

Initial tagged release for early version of filecoin

### Changelog

Initial feature set including parallel requests, selectors, basic architecture,
etc. -- changelog not tracked due to lack of go.mod

### 🙌🏽 Want to contribute?

Would you like to contribute to this repo and don’t know how? Here are a few places you can get started:

- Check out the [Contributing Guidelines](https://github.com/ipfs/go-graphsync/blob/master/CONTRIBUTING.md)
- Look for issues with the `good-first-issue` label in [go-graphsync](https://github.com/ipfs/go-graphsync/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3A%22e-good-first-issue%22+)
