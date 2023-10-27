***go-graphsync's default branch is now `main`***

To update your local repository run:
```
git branch -m master main
git fetch origin
git branch -u origin/main main
git remote set-head origin -a
```

# go-graphsync

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![Matrix](https://img.shields.io/badge/matrix-%23ipfs%3Amatrix.org-blue.svg?style=flat-square)](https://matrix.to/#/#ipfs:matrix.org)
[![IRC](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Discord](https://img.shields.io/discord/475789330380488707?color=blueviolet&label=discord&style=flat-square)](https://discord.gg/24fmuwR)
[![Coverage Status](https://codecov.io/gh/ipfs/go-graphsync/branch/master/graph/badge.svg)](https://codecov.io/gh/ipfs/go-graphsync/branch/master)
[![Build Status](https://circleci.com/gh/ipfs/go-bitswap.svg?style=svg)](https://circleci.com/gh/ipfs/go-graphsync)

> An implementation of the [graphsync protocol](https://github.com/ipld/specs/blob/master/block-layer/graphsync/graphsync.md) in go!

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Usage](#usage)
- [Architecture](#architecture)
- [Contribute](#contribute)
- [License](#license)

## Background

[GraphSync](https://github.com/ipld/specs/blob/master/block-layer/graphsync/graphsync.md) is a protocol for synchronizing IPLD graphs among peers. It allows a host to make a single request to a remote peer for all of the results of traversing an [IPLD selector](https://ipld.io/specs/selectors/) on the remote peer's local IPLD graph. 

`go-graphsync` provides an implementation of the Graphsync protocol in go.

### Go-IPLD-Prime

`go-graphsync` relies on `go-ipld-prime` to traverse IPLD Selectors in an IPLD graph. `go-ipld-prime` implements the [IPLD specification](https://github.com/ipld/specs) in go and is an alternative to older implementations such as `go-ipld-format` and `go-ipld-cbor`. In order to use `go-graphsync`, some understanding and use of `go-ipld-prime` concepts is necessary. 

If your existing library (i.e. `go-ipfs` or `go-filecoin`) uses these other older libraries, you can largely use go-graphsync without switching to `go-ipld-prime` across your codebase, but it will require some translations

## Install

`go-graphsync` requires Go >= 1.13 and can be installed using Go modules

## Usage

### Initializing a GraphSync Exchange

```golang
import (
  graphsync "github.com/ipfs/go-graphsync/impl"
  gsnet "github.com/ipfs/go-graphsync/network"
  ipld "github.com/ipld/go-ipld-prime"
)

var ctx context.Context
var host libp2p.Host
var lsys ipld.LinkSystem

network := gsnet.NewFromLibp2pHost(host)
exchange := graphsync.New(ctx, network, lsys)
```

Parameter Notes:

1. `context` is just the parent context for all of GraphSync
2. `network` is a network abstraction provided to Graphsync on top
of libp2p. This allows graphsync to be tested without the actual network
3. `lsys` is an go-ipld-prime LinkSystem, which provides mechanisms loading and constructing go-ipld-prime nodes from a link, and saving ipld prime nodes to serialized data

### Using GraphSync With An IPFS BlockStore

GraphSync provides a convenience function in the `storeutil` package for
integrating with BlockStore's from IPFS.

```golang
import (
  graphsync "github.com/ipfs/go-graphsync/impl"
  gsnet "github.com/ipfs/go-graphsync/network"
  storeutil "github.com/ipfs/go-graphsync/storeutil"
  ipld "github.com/ipld/go-ipld-prime"
  blockstore "github.com/ipfs/go-ipfs-blockstore"
)

var ctx context.Context
var host libp2p.Host
var bs blockstore.Blockstore

network := gsnet.NewFromLibp2pHost(host)
lsys := storeutil.LinkSystemForBlockstore(bs)

exchange := graphsync.New(ctx, network, lsys)
```

### Calling Graphsync

```golang
var exchange graphsync.GraphSync
var ctx context.Context
var p peer.ID
var selector ipld.Node
var rootLink ipld.Link

var responseProgress <-chan graphsync.ResponseProgress
var errors <-chan error

responseProgress, errors = exchange.Request(ctx context.Context, p peer.ID, root ipld.Link, selector ipld.Node)
```

Paramater Notes:
1. `ctx` is the context for this request. To cancel an in progress request, cancel the context.
2. `p` is the peer you will send this request to
3. `link` is an IPLD Link, i.e. a CID (cidLink.Link{Cid})
4. `selector` is an IPLD selector node. Recommend using selector builders from go-ipld-prime to construct these

### Response Type

```golang

type ResponseProgress struct {
  Node      ipld.Node // a node which matched the graphsync query
  Path      ipld.Path // the path of that node relative to the traversal start
	LastBlock struct {  // LastBlock stores the Path and Link of the last block edge we had to load. 
		ipld.Path
		ipld.Link
	}
}

```

The above provides both immediate and relevant metadata for matching nodes in a traversal, and is very similar to the information provided by a local IPLD selector traversal in `go-ipld-prime`

## Contribute

PRs are welcome!

Before doing anything heavy, checkout the [Graphsync Architecture](docs/architecture.md)

See our [Contributing Guidelines](https://github.com/ipfs/go-graphsync/blob/master/CONTRIBUTING.md) for more info.

## License

This library is dual-licensed under Apache 2.0 and MIT terms.

Copyright 2019. Protocol Labs, Inc.
