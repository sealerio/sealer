go-ipld-cbor
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Coverage Status](https://coveralls.io/repos/github/libp2p/js-libp2p-floodsub/badge.svg?branch=master)](https://coveralls.io/github/libp2p/js-libp2p-floodsub?branch=master)
[![Travis CI](https://travis-ci.org/libp2p/js-libp2p-floodsub.svg?branch=master)](https://travis-ci.org/libp2p/js-libp2p-floodsub)

> An implementation of a cbor encoded merkledag object.

## Status

This library **has alternatives available**: For new projects, prefer using the [cbor codec](https://github.com/ipld/go-ipld-prime/tree/master/codec/dagcbor) included with [go-ipld-prime](https://github.com/ipld/go-ipld-prime).

This library is in **standby** mode.  It works, but we recommend migrating to alternatives if possible.  New features are unlikely to be added here.

## Lead Maintainer

[Eric Myhre](https://github.com/warpfork)

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [API](#api)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
make install
```

## Usage

Note: This package isn't the easiest to use.
```go
// Make an object
obj := map[interface{}]interface{}{
	"foo": "bar",
	"baz": &Link{
		Target: myCid,
	},
}

// Parse it into an ipldcbor node
nd, err := WrapMap(obj)

fmt.Println(nd.Links())

```

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson
