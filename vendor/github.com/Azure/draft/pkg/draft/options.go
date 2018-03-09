package draft

// ClientOpt is an optional draft client configuration.
type ClientOpt func(*clientOpts)

type clientOpts struct {
	logLimit int64
}

func defaultClientOpts() *clientOpts {
	return &clientOpts{
		logLimit: 0,
	}
}

// WithLogsLimit sets an upper bound on the number of log lines returned
// from a draft logs client request.
func WithLogsLimit(limit int64) ClientOpt {
	return func(opts *clientOpts) {
		if limit > 0 {
			opts.logLimit = limit
		}
	}
}
