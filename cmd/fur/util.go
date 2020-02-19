package main

import (
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
