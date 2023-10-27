# go-fs-lock

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/ipfs/go-fs-lock?status.svg)](https://godoc.org/github.com/ipfs/go-fs-lock)
[![Coverage Status](https://coveralls.io/repos/github/ipfs/go-fs-lock/badge.svg?branch=master)](https://coveralls.io/github/ipfs/go-fs-lock?branch=master)
[![Travis CI](https://travis-ci.org/ipfs/go-fs-lock.svg?branch=master)](https://travis-ci.org/ipfs/go-fs-lock)

> Filesystem based locking

## Lead Maintainer

[Steven Allen](https://github.com/Stebalien)


## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-fs-lock` is a standard Go module which can be installed with:

```sh
go get github.com/ipfs/go-fs-lock
```

Note that `go-fs-lock` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).

## Usage

### Using Gx and Gx-go

This module is packaged with [Gx](https://github.com/whyrusleeping/gx). In order to use it in your own project it is recommended that you:

```sh
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/ipfs/go-fs-lock
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.

### Running tests

Before running tests, please run:

```sh
make deps
```

This will make sure that dependencies are rewritten to known working versions.

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Protocol Labs, Inc.
