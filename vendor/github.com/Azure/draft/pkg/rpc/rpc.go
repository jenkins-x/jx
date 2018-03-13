package rpc

import (
	"net"

	"github.com/Azure/draft/pkg/version"
	"golang.org/x/net/context"
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

// LogsHandler is the mechanism by which draft build logs requests
// initiated by the draft client are dispatched by the rpc.Server.
type LogsHandler interface {
	// Logs is the handler to fetch logs for a draft build.
	Logs(context.Context, *GetLogsRequest) (*GetLogsResponse, error)
}

// UpHandler is the mechanism by which to accept draft up requests
// initiated by the draft client dispatched by the rpc.Server.
type UpHandler interface {
	Up(context.Context, *UpRequest) <-chan *UpSummary
}

// Handler is the api surface to the rpc package. When calling
// Server.Server, requests are dispatched specific embedded
// interfaces within Handler.
type Handler interface {
	LogsHandler
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
		GetLogs(context.Context, *GetLogsRequest) (*GetLogsResponse, error)
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
