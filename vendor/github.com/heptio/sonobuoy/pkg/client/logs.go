/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	bufSize = 4096
)

// Reader provides an io.Reader interface to a channel of bytes. The first error
// received on the error channel will be returned by Read after the bytestream
// is drained and on all subsequent calls to Read. It is the responsibility of
// the program writing to bytestream to write an io.EOF to the error stream when
// it is done and close all channels.
type Reader struct {
	bytestream chan []byte
	errc       chan error
	done       chan struct{}

	// Used for when one message is too large for the input buffer
	// TODO(chuckha) consider alternative data structures here
	overflowBuffer []byte
	err            error
}

// NewReader returns a configured Reader.
func NewReader(bytestream chan []byte, errc chan error) *Reader {
	reader := &Reader{
		bytestream:     bytestream,
		errc:           errc,
		overflowBuffer: []byte{},
		err:            nil,
	}
	return reader
}

// Read tries to fill up the passed in byte slice with messages from the channel.
// Read manages the message overflow ensuring no bytes are missed.
// If an error is set on the reader it will return the error immediately.
func (r *Reader) Read(p []byte) (int, error) {
	// Always return the error if it is set.
	if r.err != nil {
		return 0, r.err
	}

	// Send any overflow before grabbing new messages.
	if len(r.overflowBuffer) > 0 {
		// If we need to chunk it, copy as much as we can and reduce the overflow buffer.
		if len(r.overflowBuffer) > len(p) {
			copy(p, r.overflowBuffer[:len(p)])
			r.overflowBuffer = r.overflowBuffer[len(p):]
			return len(p), nil
		}
		// At this point the entire overflow will fit into the buffer.
		copy(p, r.overflowBuffer)
		n := len(r.overflowBuffer)
		r.overflowBuffer = nil
		return n, nil
	}

	data, ok := <-r.bytestream
	// If the bytestream is done then save the error for future calls to Read.
	if !ok {
		r.err = <-r.errc
		return 0, r.err
	}

	// TODO(chuckha) this code and the code above in the overflow buffer is identical. Might be an indication of a cleaner way to do this.

	// The incoming data is bigger than size of the remaining size of the buffer. Save overflow data for next read.
	if len(data) > len(p) {
		copy(p, data[:len(p)])
		r.overflowBuffer = data[len(p):]
		return len(p), nil
	}

	// We have enough headroom in the buffer, copy all of it.
	copy(p, data)
	return len(data), nil
}

// LogReader configures a Reader that provides an io.Reader interface to a merged stream of logs from various containers.
func (s *SonobuoyClient) LogReader(cfg *LogConfig) (*Reader, error) {
	client, err := s.Client()
	if err != nil {
		return nil, err
	}
	pods, err := client.CoreV1().Pods(cfg.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pods")
	}

	errc := make(chan error)
	agg := make(chan *message)
	var wg sync.WaitGroup

	// TODO(chuckha) if we get an error back that the container is still creating maybe we could retry?
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			wg.Add(1)
			ls := &logStreamer{
				ns:        pod.Namespace,
				pod:       pod.Name,
				container: container.Name,
				errc:      errc,
				logc:      agg,
				logOpts: &v1.PodLogOptions{
					Container: container.Name,
					Follow:    cfg.Follow,
				},
				client: client,
			}

			go func(w *sync.WaitGroup, ls *logStreamer) {
				defer w.Done()
				ls.stream()
			}(&wg, ls)
		}
	}

	// Cleanup when finished.
	go func(wg *sync.WaitGroup, agg chan *message, errc chan error) {
		wg.Wait()
		close(agg)
		errc <- io.EOF
		close(errc)
	}(&wg, agg, errc)

	return NewReader(applyHeaders(agg), errc), nil
}

// message represents a buffer of logs from a container in a pod in a namespace.
type message struct {
	// preamble acts as the id for a particular container as well as the data to print before the actual logs.
	preamble string
	// buffer is the blob of logs that we extracted from the container.
	buffer []byte
}

func newMessage(preamble string, data []byte) *message {
	// Copy the bytes out of data so that byte slice can be reused.
	d := make([]byte, len(data))
	copy(d, data)
	return &message{
		preamble: preamble,
		buffer:   d,
	}
}

// logStreamer writes logs from a container to a fan-in channel.
type logStreamer struct {
	ns, pod, container string
	errc               chan error
	logc               chan *message
	logOpts            *v1.PodLogOptions
	client             kubernetes.Interface
}

// stream will open a connection to the pod's logs and push messages onto a fan-in channel.
func (l *logStreamer) stream() {
	req := l.client.CoreV1().Pods(l.ns).GetLogs(l.pod, l.logOpts)
	readCloser, err := req.Stream()
	if err != nil {
		l.errc <- errors.Wrapf(err, "error streaming logs from container [%v]", l.container)
		return
	}
	defer readCloser.Close()

	// newline because logs have new lines in them
	preamble := fmt.Sprintf("namespace=%q pod=%q container=%q\n", l.ns, l.pod, l.container)

	buf := make([]byte, bufSize)
	// Loop until EOF (streaming case won't get an EOF)
	for {
		n, err := readCloser.Read(buf)
		if err != nil && err != io.EOF {
			l.errc <- errors.Wrapf(err, "error reading logs from container [%v]", l.container)
			return
		}
		if n > 0 {
			l.logc <- newMessage(preamble, buf[:n])
		}
		if err == io.EOF {
			return
		}
	}
}

// applyHeaders takes a channel of messages and transforms it into a channel of bytes.
// applyHeaders will write headers to the byte stream as appropriate.
func applyHeaders(mesc chan *message) chan []byte {
	out := make(chan []byte)
	go func() {
		header := ""
		for message := range mesc {
			// Add the header if the header is different (ie the message is coming from a different source)
			if message.preamble != header {
				out <- []byte(message.preamble)
				header = message.preamble
			}
			out <- message.buffer
		}
		close(out)
	}()
	return out
}
