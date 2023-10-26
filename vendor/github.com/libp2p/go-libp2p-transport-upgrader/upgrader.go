package stream

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ipnet "github.com/libp2p/go-libp2p-core/pnet"
	"github.com/libp2p/go-libp2p-core/sec"
	"github.com/libp2p/go-libp2p-core/transport"
	pnet "github.com/libp2p/go-libp2p-pnet"
	manet "github.com/multiformats/go-multiaddr/net"
)

// ErrNilPeer is returned when attempting to upgrade an outbound connection
// without specifying a peer ID.
var ErrNilPeer = errors.New("nil peer")

// AcceptQueueLength is the number of connections to fully setup before not accepting any new connections
var AcceptQueueLength = 16

// Upgrader is a multistream upgrader that can upgrade an underlying connection
// to a full transport connection (secure and multiplexed).
type Upgrader struct {
	PSK       ipnet.PSK
	Secure    sec.SecureMuxer
	Muxer     mux.Multiplexer
	ConnGater connmgr.ConnectionGater
}

// UpgradeListener upgrades the passed multiaddr-net listener into a full libp2p-transport listener.
func (u *Upgrader) UpgradeListener(t transport.Transport, list manet.Listener) transport.Listener {
	ctx, cancel := context.WithCancel(context.Background())
	l := &listener{
		Listener:  list,
		upgrader:  u,
		transport: t,
		threshold: newThreshold(AcceptQueueLength),
		incoming:  make(chan transport.CapableConn),
		cancel:    cancel,
		ctx:       ctx,
	}
	go l.handleIncoming()
	return l
}

// UpgradeOutbound upgrades the given outbound multiaddr-net connection into a
// full libp2p-transport connection.
// Deprecated: use Upgrade instead.
func (u *Upgrader) UpgradeOutbound(ctx context.Context, t transport.Transport, maconn manet.Conn, p peer.ID) (transport.CapableConn, error) {
	return u.Upgrade(ctx, t, maconn, network.DirOutbound, p)
}

// UpgradeInbound upgrades the given inbound multiaddr-net connection into a
// full libp2p-transport connection.
// Deprecated: use Upgrade instead.
func (u *Upgrader) UpgradeInbound(ctx context.Context, t transport.Transport, maconn manet.Conn) (transport.CapableConn, error) {
	return u.Upgrade(ctx, t, maconn, network.DirInbound, "")
}

// Upgrade upgrades the multiaddr/net connection into a full libp2p-transport connection.
func (u *Upgrader) Upgrade(ctx context.Context, t transport.Transport, maconn manet.Conn, dir network.Direction, p peer.ID) (transport.CapableConn, error) {
	if dir == network.DirOutbound && p == "" {
		return nil, ErrNilPeer
	}
	var stat network.Stat
	if cs, ok := maconn.(network.ConnStat); ok {
		stat = cs.Stat()
	}

	var conn net.Conn = maconn
	if u.PSK != nil {
		pconn, err := pnet.NewProtectedConn(u.PSK, conn)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to setup private network protector: %s", err)
		}
		conn = pconn
	} else if ipnet.ForcePrivateNetwork {
		log.Error("tried to dial with no Private Network Protector but usage of Private Networks is forced by the environment")
		return nil, ipnet.ErrNotInPrivateNetwork
	}

	sconn, server, err := u.setupSecurity(ctx, conn, p, dir)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to negotiate security protocol: %s", err)
	}

	// call the connection gater, if one is registered.
	if u.ConnGater != nil && !u.ConnGater.InterceptSecured(dir, sconn.RemotePeer(), maconn) {
		if err := maconn.Close(); err != nil {
			log.Errorf("failed to close connection with peer %s and addr %s; err: %s",
				p.Pretty(), maconn.RemoteMultiaddr(), err)
		}
		return nil, fmt.Errorf("gater rejected connection with peer %s and addr %s with direction %d",
			sconn.RemotePeer().Pretty(), maconn.RemoteMultiaddr(), dir)
	}

	smconn, err := u.setupMuxer(ctx, sconn, server)
	if err != nil {
		sconn.Close()
		return nil, fmt.Errorf("failed to negotiate stream multiplexer: %s", err)
	}

	tc := &transportConn{
		MuxedConn:      smconn,
		ConnMultiaddrs: maconn,
		ConnSecurity:   sconn,
		transport:      t,
		stat:           stat,
	}
	return tc, nil
}

func (u *Upgrader) setupSecurity(ctx context.Context, conn net.Conn, p peer.ID, dir network.Direction) (sec.SecureConn, bool, error) {
	if dir == network.DirInbound {
		return u.Secure.SecureInbound(ctx, conn, p)
	}
	return u.Secure.SecureOutbound(ctx, conn, p)
}

func (u *Upgrader) setupMuxer(ctx context.Context, conn net.Conn, server bool) (mux.MuxedConn, error) {
	// TODO: The muxer should take a context.
	done := make(chan struct{})

	var smconn mux.MuxedConn
	var err error
	go func() {
		defer close(done)
		smconn, err = u.Muxer.NewConn(conn, server)
	}()

	select {
	case <-done:
		return smconn, err
	case <-ctx.Done():
		// interrupt this process
		conn.Close()
		// wait to finish
		<-done
		return nil, ctx.Err()
	}
}
