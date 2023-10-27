package namesys

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/libp2p/go-msgio/protoio"

	pb "github.com/libp2p/go-libp2p-pubsub-router/pb"

	"github.com/gogo/protobuf/proto"
)

const FetchProtoID = protocol.ID("/libp2p/fetch/0.0.1")

type fetchProtocol struct {
	ctx  context.Context
	host host.Host
}

type getValue func(ctx context.Context, key string) ([]byte, error)

func newFetchProtocol(ctx context.Context, host host.Host, getData getValue) *fetchProtocol {
	p := &fetchProtocol{ctx, host}

	host.SetStreamHandler(FetchProtoID, func(s network.Stream) {
		p.receive(s, getData)
	})

	return p
}

func (p *fetchProtocol) receive(s network.Stream, getData getValue) {
	defer s.Close()

	msg := &pb.FetchRequest{}
	if err := readMsg(p.ctx, s, msg); err != nil {
		log.Infof("error reading request from %s: %s", s.Conn().RemotePeer(), err)
		s.Reset()
		return
	}

	response, err := getData(p.ctx, msg.Identifier)
	var respProto pb.FetchResponse

	if err != nil {
		respProto = pb.FetchResponse{Status: pb.FetchResponse_NOT_FOUND}
	} else {
		respProto = pb.FetchResponse{Data: response}
	}

	if err := writeMsg(p.ctx, s, &respProto); err != nil {
		s.Reset()
		return
	}
}

func (p *fetchProtocol) Fetch(ctx context.Context, pid peer.ID, key string) ([]byte, error) {
	peerCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	s, err := p.host.NewStream(peerCtx, pid, FetchProtoID)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	msg := &pb.FetchRequest{Identifier: key}

	if err := writeMsg(ctx, s, msg); err != nil {
		_ = s.Reset()
		return nil, err
	}

	if err := s.CloseWrite(); err != nil {
		_ = s.Reset()
		return nil, err
	}

	response := &pb.FetchResponse{}
	if err := readMsg(ctx, s, response); err != nil {
		_ = s.Reset()
		return nil, err
	}

	switch response.Status {
	case pb.FetchResponse_OK:
		return response.Data, nil
	case pb.FetchResponse_NOT_FOUND:
		return nil, nil
	default:
		return nil, errors.New("fetch: received unknown status code")
	}
}

func writeMsg(ctx context.Context, s network.Stream, msg proto.Message) error {
	done := make(chan error, 1)
	go func() {
		wc := protoio.NewDelimitedWriter(s)

		if err := wc.WriteMsg(msg); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	var retErr error
	select {
	case retErr = <-done:
	case <-ctx.Done():
		retErr = ctx.Err()
	}

	if retErr != nil {
		log.Infof("error writing response to %s: %s", s.Conn().RemotePeer(), retErr)
	}
	return retErr
}

func readMsg(ctx context.Context, s network.Stream, msg proto.Message) error {
	done := make(chan error, 1)
	go func() {
		r := protoio.NewDelimitedReader(s, 1<<20)
		if err := r.ReadMsg(msg); err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		s.Reset()
		return ctx.Err()
	}
}
