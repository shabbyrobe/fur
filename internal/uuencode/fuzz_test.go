package uuencode

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var splitNums = regexp.MustCompile(`\s*[ ,]\s*`)

func parseInts(s string) (out []int) {
	for _, n := range splitNums.Split(strings.TrimSpace(s), -1) {
		v, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		out = append(out, v)
	}
	return out
}

func TestRandomWriteRead(t *testing.T) {
	max := 2049
	var ioSizes, dataSizes []int

	if os.Getenv("UU_IO_SIZES") != "" {
		ioSizes = parseInts(os.Getenv("UU_IO_SIZES"))

	} else {
		ioSizes = append(ioSizes, max)
		for i, jump := 1, 1; i < max; i += jump {
			ioSizes = append(ioSizes, i)
			if i > 64 {
				jump = 7
			} else if i > 128 {
				jump = 13
			}
		}
	}

	if os.Getenv("UU_DATA_SIZES") != "" {
		dataSizes = parseInts(os.Getenv("UU_DATA_SIZES"))

	} else {
		dataSizes = append(dataSizes, max)
		for i := 1; i < max; i++ {
			dataSizes = append(dataSizes, i)
		}
	}

	bts := make([]byte, max)
	rand.Read(bts)

	var buf bytes.Buffer
	var scratch = make([]byte, 2048)

	var readScratch = make([]byte, max)
	var readDest = make([]byte, max)

	for _, datasz := range dataSizes {
		for _, iosz := range ioSizes {
			if iosz > max {
				iosz = max
			}
			t.Log(fmt.Sprintf("datasz=%d,iosz=%d", datasz, iosz))

			buf.Reset()
			in := bts[:datasz]
			w := NewWriter(&buf, "-", 0644)
			tiosz := iosz
			for i := 0; i < datasz; i += iosz {
				if i+tiosz > datasz {
					tiosz = datasz - i
				}

				n, err := w.Write(in[i : i+tiosz])
				if err != nil {
					t.Fatal(err)
				}
				if n != tiosz {
					t.Fatal(n, "!=", tiosz)
				}
			}
			if err := w.Flush(); err != nil {
				t.Fatal(err)
			}

			r := NewReader(&buf, scratch)
			bts := readDest[:0]
			for {
				n, err := r.Read(readScratch[:iosz])
				if err != nil && err != io.EOF {
					t.Fatal(err)
				} else if n == 0 && err == io.EOF {
					break
				}
				bts = append(bts, readScratch[:n]...)
			}

			if !bytes.Equal(in, bts) {
				for i := 0; i < len(in) && i < len(bts); i++ {
					if in[i] != bts[i] {
						t.Fatal("first mismatched byte:", i, "/", datasz)
					}
				}
				t.Fatal("bytes do not match")
			}
		}
	}
}
