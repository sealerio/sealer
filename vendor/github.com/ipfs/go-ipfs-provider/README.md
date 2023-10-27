# go-ipfs-provider

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Coverage Status](https://codecov.io/gh/ipfs/go-ipfs-provider/branch/master/graph/badge.svg)](https://codecov.io/gh/ipfs/go-ipfs-provider)
[![Travis CI](https://travis-ci.org/ipfs/go-ipfs-provider.svg?branch=master)](https://travis-ci.org/ipfs/go-ipfs-provider)

## Background

The provider system is responsible for announcing and reannouncing to the ipfs network that a node has content.

## Install

Via `go get`:

```sh
$ go get github.com/ipfs/go-ipfs-provider
```

> Requires Go 1.12

## Usage

Here's how you create, start, interact with, and stop the provider system:

```golang
import (
	"context"
	"time"

	"github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
)

rsys := (your routing system here)
dstore := (your datastore here)
cid := (your cid to provide here)

q := queue.NewQueue(context.Background(), "example", dstore)

reprov := simple.NewReprovider(context.Background(), time.Hour * 12, rsys, simple.NewBlockstoreProvider(dstore))
prov := simple.NewProvider(context.Background(), q, rsys)
sys := provider.NewSystem(prov, reprov)

sys.Run()

sys.Provide(cid)

sys.Close()
```

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

This library is dual-licensed under Apache 2.0 and MIT terms.

Copyright 2019. Protocol Labs, Inc.
