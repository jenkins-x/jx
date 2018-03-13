package rpc

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// clientOpts specifies the union of all configurable
// options an rpc.Client accepts to communicate with
// the draft rpc.Server.
type clientOpts struct {
	dialOpts []grpc.DialOption
	addr     string
	host     string
}

// WithServerAddr sets the draft server address
// the client should dial when invoking an rpc.
func WithServerAddr(addr string) ClientOpt {
	return func(opts *clientOpts) {
		opts.addr = addr
	}
}

// WithServerHost sets the draft server host
// the client should use when invoking an rpc.
func WithServerHost(host string) ClientOpt {
	return func(opts *clientOpts) {
		opts.host = host
	}
}

// WithGrpcDialOpt adds the provided grpc.DialOptions for the
// rpc.Client to use when initializing the underlying grpc.Client.
func WithGrpcDialOpt(dialOpts ...grpc.DialOption) ClientOpt {
	return func(opts *clientOpts) {
		opts.dialOpts = append(opts.dialOpts, dialOpts...)
	}
}

// DefaultClientOpts returns the set of default rpc ClientOpts the draft
// client requires.
func DefaultClientOpts() []ClientOpt {
	return []ClientOpt{
		WithGrpcDialOpt(
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time: time.Duration(30) * time.Second,
			}),
			grpc.WithTimeout(5*time.Second),
			grpc.WithBlock(),
		),
	}
}

// serverOpts specifies the union of all configurable
// options an rpc.Server accepts, e.g. TLS config.
type serverOpts struct {
	grpcOpts []grpc.ServerOption
}

// WithGrpcServerOpt adds the provided grpc.ServerOption
// for the rpc.Server to use when initializing the underlying
// grpc.Server.
func WithGrpcServerOpt(grpcOpts ...grpc.ServerOption) ServerOpt {
	return func(opts *serverOpts) {
		opts.grpcOpts = append(opts.grpcOpts, grpcOpts...)
	}
}

// DefaultServerOpts returns the set of default rpc ServerOpts that draftd requires.
func DefaultServerOpts() []ServerOpt {
	return []ServerOpt{
		WithGrpcServerOpt(grpc.MaxMsgSize(maxMsgSize)),
	}
}
