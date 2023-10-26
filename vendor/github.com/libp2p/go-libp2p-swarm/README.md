go-libp2p-swarm
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/libp2p/go-libp2p-swarm)](https://pkg.go.dev/github.com/libp2p/go-libp2p-swarm)
[![](https://img.shields.io/badge/project-libp2p-yellow.svg?style=flat-square)](https://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23libp2p-yellow.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23libp2p)
[![Code Coverage](https://img.shields.io/codecov/c/github/libp2p/go-libp2p-swarm/master.svg?style=flat-square)](https://codecov.io/gh/libp2p/go-libp2p-swarm)
[![Discourse posts](https://img.shields.io/discourse/https/discuss.libp2p.io/posts.svg)](https://discuss.libp2p.io)

> The libp2p swarm manages groups of connections to peers, and handles incoming and outgoing streams.

The libp2p swarm is the 'low level' interface for working with a given libp2p
network. It gives you more fine grained control over various aspects of the
system. Most applications don't need this level of access, so the `Swarm` is
generally wrapped in a `Host` abstraction that provides a more friendly
interface. See [the host interface](https://godoc.org/github.com/libp2p/go-libp2p-core/host#Host)
for more info on that.

## Table of Contents

- [Install](#install)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
go get github.com/libp2p/go-libp2p-swarm
```

## Usage

### Creating a swarm

To construct a swarm, you'll be calling `NewSwarm`. That function looks like this:
```go
swarm, err := NewSwarm(peerID, peerstore)
```

The first parameter of the swarm constructor is an identity in the form of a peer.ID.

The second argument is a peerstore. This is essentially a database that the
swarm will use to store peer IDs, addresses, public keys, protocol preferences
and more.

### Streams
The swarm is designed around using multiplexed streams to communicate with
other peers. When working with a swarm, you will want to set a function to
handle incoming streams from your peers:

```go
swrm.SetStreamHandler(func(s network.Stream) {
	defer s.Close()
	fmt.Println("Got a stream from: ", s.SwarmConn().RemotePeer())
	fmt.Fprintln(s, "Hello Friend!")
})
```

Tip: Always make sure to close streams when you're done with them.


## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson

---

The last gx published version of this module was: 3.0.35: QmQVoMEL1CxrVusTSUdYsiJXVBnvSqNUpBsGybkwSfksEF
