package grpcclient

import (
	"context"
	"io"
	"sync"

	"google.golang.org/grpc"
)

type grpcStream struct {
	grpc.ClientStream

	sync.RWMutex
	closed  bool
	err     error
	conn    *poolConn
	context context.Context
	close   func(err error)
}

func (g *grpcStream) Context() context.Context {
	return g.context
}

func (g *grpcStream) Send(msg any) error {
	if err := g.ClientStream.SendMsg(msg); err != nil {
		g.setError(err)
		return err
	}
	return nil
}

func (g *grpcStream) Recv(msg any) (err error) {
	defer g.setError(err)

	if err = g.ClientStream.RecvMsg(msg); err != nil {
		closeErr := g.Close()
		if err == io.EOF && closeErr != nil {
			err = closeErr
		}
		return err
	}
	return
}

func (g *grpcStream) Error() error {
	g.RLock()
	defer g.RUnlock()
	return g.err
}

func (g *grpcStream) setError(e error) {
	g.Lock()
	g.err = e
	g.Unlock()
}

func (g *grpcStream) Close() error {
	g.Lock()
	defer g.Unlock()

	if g.closed {
		return nil
	}

	g.closed = true
	g.close(g.err)
	return g.ClientStream.CloseSend()
}
