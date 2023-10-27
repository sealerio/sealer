# go-ipfs-keystore

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![Travis CI](https://travis-ci.com/ipfs/go-ipfs-keystore.svg?branch=master)](https://travis-ci.com/ipfs/go-ipfs-keystore)


> go-ipfs-keystore implements keystores for ipfs

go-ipfs-keystore provides the Keystore interface for key management.  Keystores support adding, retrieving, and deleting keys as well as listing all keys and checking for membership.

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-ipfs-keystore` works like a regular Go module:
```
> go get github.com/ipfs/go-ipfs-keystore
```

It uses [Gx](https://github.com/whyrusleeping/gx) to manage dependencies.

## Usage
```
import "github.com/ipfs/go-ipfs-keystore"
```

## Contribute

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

This project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)
