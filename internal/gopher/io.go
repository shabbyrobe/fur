package gopher

import "io"

func writerNullCloser() error { return nil }

type writerWithCloser struct {
	writer io.Writer
	closer func() error
}

func newWriterWithCloser(wr io.Writer, closer func() error) *writerWithCloser {
	if closer == nil {
		wc, ok := wr.(io.Closer)
		if ok {
			closer = wc.Close
		} else {
			closer = writerNullCloser
		}
	}

	return &writerWithCloser{wr, closer}
}

func (wwc *writerWithCloser) Write(b []byte) (n int, err error) {
	return wwc.writer.Write(b)
}

func (wwc *writerWithCloser) Close() error {
	return wwc.closer()
}
