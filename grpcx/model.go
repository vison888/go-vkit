package grpcx

import (
	"context"

	"google.golang.org/grpc"
)

type ApiEndpoint struct {
	Method       string
	Url          string
	ClientStream bool
	ServerStream bool
}

type FileInfo struct {
	Filename string
	Size     int64
	Content  []byte
}

type ClientStream interface {
	// Context for the stream
	Context() context.Context
	// Send will encode and send a request
	Send(any) error
	// Recv will decode and read a response
	Recv(any) error
	// Error returns the stream error
	Error() error
	// Close closes the stream
	Close() error
}

type Client interface {
	Invoke(ctx context.Context, serive, endpoint string, args any, reply any, opts ...grpc.CallOption) error
	NewStream(ctx context.Context, desc *grpc.StreamDesc, serive, endpoint string, opts ...grpc.CallOption) (ClientStream, error)
}
