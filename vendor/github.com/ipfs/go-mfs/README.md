# go-mfs

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/ipfs/go-mfs?status.svg)](https://godoc.org/github.com/ipfs/go-mfs)
[![Build Status](https://travis-ci.com/ipfs/go-mfs.svg?branch=master)](https://travis-ci.com/ipfs/go-mfs)

> go-mfs implements an in-memory model of a mutable IPFS filesystem.

## Lead Maintainer

[Steven Allen](https://github.com/Stebalien)

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-mfs` works like a regular Go module:

```
> go get github.com/ipfs/go-mfs
```

It uses [Gx](https://github.com/whyrusleeping/gx) to manage dependencies. You can use `make all` to build it with the `gx` dependencies.

## Usage

```
import "github.com/ipfs/go-mfs"
```

Check the [GoDoc documentation](https://godoc.org/github.com/ipfs/go-mfs)

## Documentation

Documentation around the MFS and the Files API in general around IPFS is a work in progress the following links may be of use:

* [UnixFS](https://docs.ipfs.io/guides/concepts/unixfs/)
* [MFS](https://docs.ipfs.io/guides/concepts/mfs/)
* [General concept document about how are files handled in IPFS (WIP)](https://github.com/ipfs/docs/issues/133)

## Repository Structure
This repository contains many files, all belonging to the root `mfs` package.

* `file.go`: MFS `File`.
* `dir.go`: MFS `Directory`.
* `fd.go`: `FileDescriptor` used to operate on `File`s.
* `ops.go`: Functions that do not belong to either `File` nor `Directory` (although they mostly operate on them) that contain common operations to the MFS, e.g., find, move, add a file, make a directory.
* `root.go`: MFS `Root` (a `Directory` with republishing support).
* `repub.go`: `Republisher`.
* `mfs_test.go`: General tests (needs a [revision](https://github.com/ipfs/go-mfs/issues/9)).
* `repub_test.go`: Republisher-specific tests (contains only the `TestRepublisher` function).

## Contribute

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Protocol Labs, Inc.
