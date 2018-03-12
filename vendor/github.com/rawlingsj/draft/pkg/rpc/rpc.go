package rpc

import (
	"github.com/Azure/draft/pkg/version"
	"golang.org/x/net/context"
	"net"
)

// RecvStream is returned by a Client for streaming summaries
// in response to a draft up. Stop should be called on return
// to notify the stream to close.
type RecvStream interface {
	Done() <-chan struct{}
	Recv() *UpSummary
	Err() error
	Stop()
}

// Up is the mechanism by which to accept draft up requests
// initiated by the draft client dispatched by the rpc.Server.
type UpHandler interface {
	Up(context.Context, *UpRequest) <-chan *UpSummary
}

// Handler is the api surface to the rpc package. When calling
// Server.Server, requests are dispatched specific embedded
// interfaces within Handler.
type Handler interface {
	UpHandler
}

type (
	// ClientOpt is an optional configuration for a client.
	ClientOpt func(*clientOpts)

	// Client handles rpc to the Server.
	Client interface {
		Version(context.Context) (*version.Version, error)
		UpBuild(context.Context, *UpRequest, chan<- *UpSummary) error
		UpStream(context.Context, <-chan *UpRequest, chan<- *UpSummary) error
	}
)

// NewClient returns a Client configured with the provided ClientOpts.
func NewClient(opts ...ClientOpt) Client { return newClientImpl(opts...) }

type (
	// ServerOpt is an optional configuration for a server.
	ServerOpt func(*serverOpts)

	// Server handles rpc made by the client.
	Server interface {
		Serve(net.Listener, Handler) error
		Stop() bool
	}
)

// NewServer returns a Server configured with the provided ServerOpts.
func NewServer(opts ...ServerOpt) Server { return newServerImpl(opts...) }
