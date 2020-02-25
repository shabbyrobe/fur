package gopher

import (
	"bufio"
	"errors"
	"io"
	"net/textproto"
)

func NewTextReader(rdr io.Reader) io.Reader {
	dot := textproto.NewReader(bufio.NewReader(rdr)).DotReader()
	wrp := &expectedUnexpectedEOFReader{rdr: dot}
	return wrp
}

func MustFlush(f interface{ Flush() error }) {
	if err := f.Flush(); err != nil {
		panic(err)
	}
}

type readCloser struct {
	readFn  func(b []byte) (int, error)
	closeFn func() error
}

func (rc *readCloser) Read(b []byte) (int, error) {
	return rc.readFn(b)
}

func (rc *readCloser) Close() error {
	return rc.closeFn()
}

type expectedUnexpectedEOFReader struct {
	rdr     io.Reader
	replace error
}

func (r *expectedUnexpectedEOFReader) Read(b []byte) (n int, err error) {
	n, err = r.rdr.Read(b)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		// ErrUnexpectedEOF comes form textproto.DotReader. As much as I'd like it if
		// Gopher servers always sent the '.\r\n' line, most of them skip it, so without
		// the Gopher+ content length and the terminator line, we have no reliable way of
		// knowing that the response is truncated.
		//
		// There's no other way to detect a truncated response as we don't get sent the
		// content length, so it's a lot better for clients if the server does send this.
		//
		// Gopher servers: please send '.\r\n'.
		if r.replace == nil {
			err = io.EOF
		} else {
			err = r.replace
		}
	}
	return n, err
}

type nilReadCloser struct{}

func (rc nilReadCloser) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}

func (rc nilReadCloser) Close() error {
	return nil
}
