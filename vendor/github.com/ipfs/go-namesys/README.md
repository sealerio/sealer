# go-namesys

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://protocol.ai)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![Go Reference](https://pkg.go.dev/badge/github.com/ipfs/go-namesys.svg)](https://pkg.go.dev/github.com/ipfs/go-namesys)
[![Travis CI](https://travis-ci.com/ipfs/go-namesys.svg?branch=master)](https://travis-ci.com/ipfs/go-namesys)


> go-namesys provides publish and resolution support for the /ipns/ namespace 

Package namesys defines `Resolver` and `Publisher` interfaces for IPNS paths, that is, paths in the form of `/ipns/<name_to_be_resolved>`. A "resolved" IPNS path becomes an `/ipfs/<cid>` path.

Traditionally, these paths would be in the form of `/ipns/{libp2p-key}`, which references an IPNS record in a distributed `ValueStore` (usually the IPFS DHT).

Additionally, the `/ipns/` namespace can also be used with domain names that use DNSLink (`/ipns/en.wikipedia-on-ipfs.org`, see https://docs.ipfs.io/concepts/dnslink/).

The package provides implementations for all three resolvers.

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-namesys` works like a regular Go module:
```
> go get github.com/ipfs/go-namesys
```

## Usage
```
import "github.com/ipfs/go-namesys"
```

See the [Pkg.go.dev documentation](https://pkg.go.dev/github.com/ipfs/go-namesys)

## Contribute

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

This project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)
