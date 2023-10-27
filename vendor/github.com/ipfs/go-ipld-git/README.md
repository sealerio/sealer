Git ipld format
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)

> An IPLD codec for git objects allowing path traversals across the git graph.

## Table of Contents

- [Install](#install)
- [About](#about)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
go get github.com/ipfs/go-ipld-git
```

## About

This is an IPLD codec which handles git objects. Objects are transformed
into IPLD graph as detailed below. Objects are demonstrated here using both
[IPLD Schemas](https://ipld.io/docs/schemas/) and example JSON forms.

### Commit

```ipldsch
type GpgSig string

type PersonInfo struct {
  date String
  timezone String
  email String
  name String
}

type Commit struct {
  tree &Tree # see "Tree" section below
  parents [&Commit]
  message String
  author optional PersonInfo
  committer optional PersonInfo
  encoding optional String
  signature optional GpgSig
  mergetag [Tag]
  other [String]
}
```

As JSON, real data would look something like:

```json
{
  "author": {
    "date": "1503667703",
    "timezone": "+0200",
    "email": "author@mail",
    "name": "Author Name"
  },
  "committer": {
    "date": "1503667703",
    "timezone": "+0200",
    "email": "author@mail",
    "name": "Author Name"
  },
  "message": "Commit Message\n",
  "parents": [
    <LINK>, <LINK>, ...
  ],
  "tree": <LINK>
}
```

### Tag

```ipldsch
type Tag struct {
  object &Any
  type String
  tag String
  tagger PersonInfo
  message String
}
```

As JSON, real data would look something like:

```json
{
  "message": "message\n",
  "object": {
    "/": "baf4bcfg3mbz3yj3njqyr3ifdaqyfv3prei6h6bq"
  },
  "tag": "tagname",
  "tagger": {
    "date": "1503667703 +0200",
    "email": "author@mail",
    "name": "Author Name"
  },
  "type": "commit"
}
```

### Tree

```ipldsch
type Tree {String:TreeEntry}

type TreeEntry struct {
  mode String
  hash &Any
}
```

As JSON, real data would look something like:

```json
{
  "file.name": {
    "mode": "100664",
    "hash": <LINK>
  },
  "directoryname": {
    "mode": "40000",
    "hash": <LINK>
  },
  ...
}
```

### Blob

```ipldsch
type Blob bytes
```

As JSON, real data would look something like:

```json
"<base64 of 'blob <size>\0<data>'>"
```

## Lead Maintainers

* [Will Scott](https://github.com/willscott)
* [Rod Vagg](https://github.com/rvagg)

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson
