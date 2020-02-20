package gopher

import (
	"bufio"
	"errors"
	"io"
	"net/textproto"
)

func DotReader(rdr io.Reader) io.Reader {
	dot := textproto.NewReader(bufio.NewReader(rdr)).DotReader()
	wrp := &expectedUnexpectedEOFReader{rdr: dot}
	return wrp
}

type readCloser struct {
	readFn  func(b []byte) (int, error)
	closeFn func() error
}

func (rc *readCloser) Read(b []byte) (int, error) { return rc.readFn(b) }
func (rc *readCloser) Close() error               { return rc.closeFn() }

type expectedUnexpectedEOFReader struct {
	rdr io.Reader
}

func (r *expectedUnexpectedEOFReader) Read(b []byte) (n int, err error) {
	n, err = r.rdr.Read(b)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		// ErrUnexpectedEOF comes form textproto.DotReader. As much as I'd like it if
		// Gopher servers always sent the '.\r\n' line, most of them skip it, so without
		// the Gopher+ content length and the terminator line, we have no reliable way of
		// knowing that the response is truncated.
		//
		// There's also no consistency to it: Veronica2 sends it for search results, which
		// are a directory listing, so it really is expected (and a good idea) to send
		// the terminator line.
		//
		// Gopher servers: please send '.\r\n'.
		err = io.EOF
	}
	return n, err
}
