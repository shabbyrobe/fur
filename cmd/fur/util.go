package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mattn/go-tty"
	"github.com/shabbyrobe/cmdy"
)

type nopWriteCloser struct {
	io.Writer
}

func (n *nopWriteCloser) Close() error { return nil }

func stdoutOrFileWriter(output io.Writer, fileName string, allowDefaultStdout bool) (rdr io.WriteCloser, isFile bool, err error) {
	isPipe := cmdy.WriterIsPipe(output)

	if fileName == "" {
		if allowDefaultStdout || isPipe {
			return &nopWriteCloser{output}, false, nil
		} else {
			return nil, false, fmt.Errorf("would write binary data to stdout; to override, set the output file or pipe stdout somewhere")
		}
	}

	if fileName == "-" {
		return &nopWriteCloser{output}, false, nil
	}

	f, err := os.Create(fileName)
	return f, true, err
}

// DeferClose closes an io.Closer and sets the error into err if one occurs and the
// value of err is nil.
func DeferClose(err *error, closer io.Closer) {
	cerr := closer.Close()
	if *err == nil && cerr != nil {
		*err = cerr
	}
}

func termSize() (x, y int) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	x, y, err = tty.Size()
	if err != nil {
		x, y = 80, 25
	}
	return x, y
}

func copyWithLcut(out io.Writer, rs io.Reader, lcut int) error {
	var scratch = make([]byte, 8192)
	var buf = make([]byte, 0, 8192)
	var readDone = false
	var pos = 0

	for !readDone {
		var idx = -1
		for idx < 0 && !readDone {
			buf = buf[:copy(buf, buf[pos:])]
			pos = 0

			n, err := rs.Read(scratch)
			if err != nil && err != io.EOF {
				return err
			}
			buf = append(buf, scratch[:n]...)
			if err == io.EOF {
				readDone = true
			}
			idx = bytes.IndexByte(buf, '\n')
		}

	again:
		var line []byte
		if idx >= 0 {
			end := pos + idx + 1
			line = buf[pos:end]
			pos = end
		} else if readDone {
			line = buf[pos:]
		} else {
			continue
		}

		if len(line) > 0 {
			lmax := len(line)
			if lmax > 0 && line[lmax-1] == '\r' {
				lmax--
			}
			cut := lcut
			if cut > lmax {
				cut = lmax
			}
			line = line[cut:]
			out.Write(line)
		}

		if !readDone {
			idx = bytes.IndexByte(buf[pos:], '\n')
			goto again
		}
	}

	return nil
}
