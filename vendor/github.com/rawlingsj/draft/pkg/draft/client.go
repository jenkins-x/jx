package draft

import (
	"crypto/tls"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Azure/draft/pkg/build"
	"github.com/Azure/draft/pkg/rpc"
	"github.com/Azure/draft/pkg/version"
	"golang.org/x/net/context"
)

// ClientConfig stores information about the draft server and where to send messages
//  and errors out
type ClientConfig struct {
	ServerAddr string
	ServerHost string
	Stdout     io.Writer
	Stderr     io.Writer
	UseTLS     bool
	TLSConfig  *tls.Config
}

type Client struct {
	cfg *ClientConfig
	rpc rpc.Client
	res chan *rpc.UpSummary
}

// NewClient takes ClientConfig and returns a Client
func NewClient(cfg *ClientConfig) *Client {
	opts := []rpc.ClientOpt{rpc.WithServerAddr(cfg.ServerAddr)}
	switch {
	case cfg.UseTLS:
		creds := grpc.WithTransportCredentials(credentials.NewTLS(cfg.TLSConfig))
		opts = append(opts, rpc.WithGrpcDialOpt(creds))
	default:
		opts = append(opts, rpc.WithGrpcDialOpt(grpc.WithInsecure()))
	}
	return &Client{
		cfg: cfg,
		rpc: rpc.NewClient(opts...),
		res: make(chan *rpc.UpSummary, 2),
	}
}

func (c *Client) Results() <-chan *rpc.UpSummary { return c.res }

func (c *Client) Version(ctx context.Context) (*version.Version, error) {
	return c.rpc.Version(ctx)
}

func (c *Client) Up(ctx context.Context, app *build.Context) error {
	req := &rpc.UpRequest{
		Namespace:  app.Env.Namespace,
		AppName:    app.Env.Name,
		Chart:      app.Chart,
		Values:     app.Values,
		AppArchive: &rpc.AppArchive{Name: app.SrcName, Content: app.Archive},
	}
	if !app.Env.Watch {
		return c.build(ctx, app, req)
	}
	return c.stream(ctx, app, req)
}

func (c *Client) build(ctx context.Context, app *build.Context, req *rpc.UpRequest) error {
	cancelctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		msgc = make(chan *rpc.UpSummary, 1)
		errc = make(chan error, 1)
	)

	go func() {
		if err := c.rpc.UpBuild(cancelctx, req, msgc); err != nil {
			errc <- err
		}
		close(errc)
	}()
	for msgc != nil || errc != nil {
		select {
		case msg, ok := <-msgc:
			if !ok {
				msgc = nil
				close(c.res)
				cancel()
				continue
			}
			select {
			case c.res <- msg:
				// deliver / buffer to summary channel
			default:
				// ignore to avoid blocking
			}
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return fmt.Errorf("error running draft up: %v", err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (c *Client) stream(ctx context.Context, app *build.Context, req *rpc.UpRequest) error {
	cancelctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		bldc = make(chan *build.Context, 1)
		reqc = make(chan *rpc.UpRequest, 1)
		msgc = make(chan *rpc.UpSummary, 1)
		errc = make(chan error, 1)
	)

	var wg sync.WaitGroup
	wg.Add(2)

	// Buffer initial up request.
	reqc <- req

	// Start a goroutine to service the stream.
	go func() {
		if err := c.rpc.UpStream(cancelctx, reqc, msgc); err != nil {
			errc <- err
		}
		wg.Done()
	}()

	// Start a goroutine to watch for builds.
	go func() {
		errc <- app.Watch(cancelctx, bldc)
		wg.Done()
	}()

	defer func() {
		close(reqc)
		close(errc)
		wg.Wait()
	}()

	for {
		select {
		// msgc is closed, handle nil msg received
		case msg := <-msgc:
			fmt.Fprintf(c.cfg.Stdout, "\r%s: %s\n", msg.StageDesc, msg.StatusText)
		case bld := <-bldc:
			reqc <- &rpc.UpRequest{
				Namespace:  bld.Env.Namespace,
				AppName:    bld.Env.Name,
				Chart:      bld.Chart,
				Values:     bld.Values,
				AppArchive: &rpc.AppArchive{Name: bld.SrcName, Content: bld.Archive},
			}
		case err := <-errc:
			return err
		case <-ctx.Done():
			// drain the swamp.
			for range errc {
			}
			return ctx.Err()
		}
	}
}
