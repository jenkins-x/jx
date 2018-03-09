package cmdline

import (
	"github.com/fatih/color"
	"io"
	"os"
)

// Option defines a configurable cmdline interface option.
type Option func(*options)

type options struct {
	stderr io.Writer
	stdout io.Writer
}

// DefaultOpts is a convenience wrapper that enumerates and configures the set of default
//  options for the cmdline interface.
func DefaultOpts() Option {
	return func(opts *options) {
		WithStderr(os.Stderr)(opts)
		WithStdout(os.Stdout)(opts)
	}
}

// WithStderr returns an Option that sets the cmdline interface's stderr writer.
//
// Defaults to os.Stderr.
func WithStderr(w io.Writer) Option {
	return func(opts *options) {
		opts.stderr = w
	}
}

// WithStdout returns an Option that sets the cmdline interface's stdout writer.
//
// Defaults to os.Stdout.
func WithStdout(w io.Writer) Option {
	return func(opts *options) {
		opts.stdout = w
	}
}

// NoColor returns an Option that sets if the output is colorized or not.
//
// Colorized output is enabled by default.
func NoColor() Option {
	return func(opts *options) {
		color.NoColor = true
	}
}
