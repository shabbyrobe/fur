//+build gofuzz

package uuencode

import (
	"bytes"
	"io"
	"io/ioutil"
)

//go:generate go-fuzz-build -o reader-fuzz.zip -func FuzzReader

func FuzzReader(data []byte) int {
	ur := NewReader(bytes.NewReader(data), nil)
	if _, err := io.Copy(ioutil.Discard, ur); err != nil {
		return 0
	}
	return 1
}

//go:generate go-fuzz-build -o writer-fuzz.zip -func FuzzWriter

func FuzzWriter(data []byte) int {
	uw := NewWriter(ioutil.Discard, "", 0644)
	if _, err := io.Copy(uw, bytes.NewReader(data)); err != nil {
		return 0
	}
	if err := uw.Flush(); err != nil {
		return 0
	}
	return 1
}
