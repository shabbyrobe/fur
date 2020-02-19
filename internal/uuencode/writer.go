package uuencode

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

var (
	writeEnd = []byte{'`', '\n', 'e', 'n', 'd', '\n'}
)

type Writer struct {
	inner    *bufio.Writer
	file     string
	mode     os.FileMode
	ln       []byte
	lnpos    byte
	lnsz     byte
	rem      []byte
	flushed  bool
	wroteHdr bool
}

func FileWriter(w *os.File, file string, mode os.FileMode) (*Writer, error) {
	inf, err := w.Stat()
	if err != nil {
		return nil, err
	}
	return NewWriter(w, w.Name(), inf.Mode()), nil
}

func NewWriter(w io.Writer, file string, mode os.FileMode) *Writer {
	bw, ok := w.(*bufio.Writer)
	if !ok {
		bw = bufio.NewWriter(w)
	}
	return &Writer{
		inner: bw,
		file:  file,
		mode:  mode,
		ln:    make([]byte, 62),
		lnpos: 1,
		rem:   make([]byte, 0, 3),
	}
}

func (w *Writer) flushLine() error {
	lnsz := w.lnsz // doesn't include newline or length
	if lnsz == 0 {
		w.ln[0] = '`' // 0-length line
	} else {
		w.ln[0] = byte(lnsz + 32)
	}
	w.ln[w.lnpos] = '\n'
	if _, err := w.inner.Write(w.ln[:w.lnpos+1]); err != nil {
		return err
	}
	w.lnsz = 0
	w.lnpos = 1
	return nil
}

func (w *Writer) writeHdr() error {
	_, err := fmt.Fprintf(w.inner, "begin %03o %s\n", w.mode, w.file)
	w.wroteHdr = true
	return err
}

func (w *Writer) Write(b []byte) (n int, err error) {
	if w.flushed {
		return n, fmt.Errorf("uunecode: already flushed")
	}
	if !w.wroteHdr {
		if err := w.writeHdr(); err != nil {
			return n, err
		}
	}

	bsz := len(b)
	remsz := len(w.rem)
	bpos := 0

	if remsz > 0 {
		if bsz+remsz < 3 {
			w.rem = append(w.rem, b...)
			return bsz, nil

		} else {
			bpos = 3 - remsz
			out := append(w.rem, b[:bpos]...)
			n = bpos
			uupack(w.ln[w.lnpos:], out)
			w.lnpos += 4
			w.lnsz += 3
			if w.lnpos >= 61 {
				if err := w.flushLine(); err != nil {
					return n, err
				}
			}
			w.rem = w.rem[:0]
		}
	}

	bend := bsz - ((bsz - bpos) % 3)
	if bend >= 3 {
		for ; n < bend; n += 3 {
			uupack(w.ln[w.lnpos:], b[n:])
			w.lnpos += 4
			w.lnsz += 3
			if w.lnpos >= 61 {
				if err := w.flushLine(); err != nil {
					return n, err
				}
			}
		}
	}

	if bsz-bend > 0 {
		w.rem = append(w.rem, b[bend:]...)
		n += bsz - bend
	}

	return n, nil
}

func (w *Writer) Flush() error {
	if w.flushed {
		return fmt.Errorf("uunecode: already flushed")
	}
	if !w.wroteHdr {
		if err := w.writeHdr(); err != nil {
			return err
		}
	}

	remsz := len(w.rem)
	if remsz > 0 {
		switch remsz {
		case 2:
			w.rem = append(w.rem, 0)
		case 1:
			w.rem = append(w.rem, 0, 0)
		case 0:
		default:
			panic("unexpected rem size")
		}
		uupack(w.ln[w.lnpos:], w.rem)
		w.lnsz += byte(remsz)
		w.lnpos += 4
	}

	w.flushed = true
	if err := w.flushLine(); err != nil {
		return err
	}
	if _, err := w.inner.Write(writeEnd); err != nil {
		return err
	}
	return w.inner.Flush()
}

var uuTable = [64]byte{
	'`', '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?',
	'@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', '[', '\\', ']', '^', '_',
}

func uupack(into []byte, b []byte) {
	// b0 = 000000 00
	// b1 = 0000 0000
	// b2 = 00 000000
	into[0] = uuTable[b[0]>>2]
	into[1] = uuTable[(b[0]<<4|b[1]>>4)&63]
	into[2] = uuTable[(b[1]<<2|b[2]>>6)&63]
	into[3] = uuTable[b[2]&63]
}
