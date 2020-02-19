package uuencode

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	readBegin = iota
	readLineLength
	readLine
	readEnd
	readDone
)

var (
	tokMagic         = []byte("begin ")
	tokReadEnd       = []byte("end")
	tokReadEmptyLine = []byte("`")
)

type Reader struct {
	inner   io.Reader
	state   int
	file    string
	mode    os.FileMode
	scratch []byte
	buf     []byte
	line    []byte
	lineRem []byte
	linesz  int
}

func NewReader(rdr io.Reader, scratch []byte) *Reader {
	if len(scratch) < 2048 {
		scratch = make([]byte, 2048)
	}
	return &Reader{
		inner:   rdr,
		scratch: scratch,
		line:    make([]byte, 63), // line can never have more than 63 chars of decoded data
	}
}

func (r *Reader) File() (name string, ok bool) {
	if r.state < readLineLength {
		return "", false
	}
	return r.file, true
}

func (r *Reader) Mode() (mode os.FileMode, ok bool) {
	if r.state < readLineLength {
		return 0, false
	}
	return r.mode, true
}

func (r *Reader) pull() (err error) {
	newsz := copy(r.scratch, r.buf)
	n, err := r.inner.Read(r.scratch[newsz:])
	n += newsz
	r.buf = r.scratch[:n]

	if n == 0 && err == io.EOF {
		if r.state != readDone {
			return r.fail("premature end of data stream")
		}
		r.state = readDone

	} else if err == io.EOF {
		err = nil
	}
	return err
}

func (r *Reader) fail(msg string) (err error) {
	r.state = readDone
	return fmt.Errorf("uudecode: %s", msg)
}

func (r *Reader) Read(b []byte) (n int, err error) {
	bsz := len(b)

	if len(r.lineRem) > 0 {
		n += copy(b, r.lineRem)
		r.lineRem = r.lineRem[n:]
		if n <= bsz {
			return n, nil
		}
	}

	for {
		switch r.state {
		case readBegin:
			if err := r.pull(); err != nil {
				return n, err
			}
			nl := bytes.IndexByte(r.scratch, '\n')
			if nl < 0 {
				return n, r.fail("header missing delimiter")
			}
			if !bytes.HasPrefix(r.scratch, tokMagic) {
				return n, r.fail("header missing magic")
			}
			pos := len(tokMagic)

			// tokMagic includes first space, so spc is the mode string's end delimiter:
			spc := bytes.IndexByte(r.scratch[pos:], ' ')
			if spc < 0 {
				return n, r.fail("missing file name")
			}

			modestr := string(r.scratch[pos : pos+spc])
			mode, err := strconv.ParseInt(modestr, 8, 16)
			if err != nil {
				return n, r.fail("invalid file mode")
			}

			r.mode = os.FileMode(mode)
			r.file = string(r.scratch[pos+spc+1 : nl])

			r.state = readLineLength
			r.buf = r.buf[nl+1:]

		case readLineLength:
			if len(r.buf) < 1 {
				if err := r.pull(); err != nil {
					return n, err
				}
				if len(r.buf) < 1 {
					return n, r.fail("missing line length")
				}
			}

			// This also works if the zero is a '`' as (96 - ' ') & 63 == 0
			r.linesz = int((r.buf[0] - ' ') & 63)
			r.buf = r.buf[1:]
			if r.linesz == 0 {
				r.state = readEnd
			} else {
				r.state = readLine
			}

		case readLine:
			nl := bytes.IndexByte(r.buf, '\n')
			if nl < 0 {
				if err := r.pull(); err != nil {
					return n, err
				}
				nl = bytes.IndexByte(r.buf, '\n')
				if nl < 0 {
					return n, r.fail("missing line ending")
				}
			}

			end := nl
			if nl > 0 && r.buf[nl-1] == '\r' {
				nl--
			}

			if nl&3 != 0 {
				return n, r.fail("encoded line length must be divisible by 4")
			}
			if nl > 64 {
				return n, r.fail("encoded line too long")
			}

			expectedDec := (nl / 4) * 3
			if expectedDec < r.linesz {
				return n, r.fail("unexpected decoded size for line")
			}

			var lenc, ldec []byte = nil, r.line
			var ienc, idec int
			lenc, r.buf = r.buf[:nl], r.buf[end+1:]
			r.state = readLineLength

			for ienc < nl {
				e0, e1, e2, e3 := lenc[ienc]-' ', lenc[ienc+1]-' ', lenc[ienc+2]-' ', lenc[ienc+3]-' '
				if e0 > 64 || e1 > 64 || e2 > 64 || e3 > 64 {
					return n, r.fail("unexpected encoded byte")
				}
				// bytes may be '64' instead of '0':
				e0, e1, e2, e3 = e0&63, e1&63, e2&63, e3&63

				ldec[idec+0] = e0<<2 | e1>>4 // dec: 111111 112222 222233 333333
				ldec[idec+1] = e1<<4 | e2>>2 // enc: 111111 222222 333333 444444
				ldec[idec+2] = e2<<6 | e3
				ienc, idec = ienc+4, idec+3
			}

			ldec = ldec[:r.linesz]
			cp := copy(b[n:], ldec)
			n += cp
			r.lineRem = ldec[cp:]
			if n == bsz {
				return n, nil
			}

		case readEnd:
			nl := bytes.IndexByte(r.buf, '\n')
			if nl < 0 {
				if err := r.pull(); err != nil {
					return n, err
				}
				nl = bytes.IndexByte(r.buf, '\n')
				if nl < 0 {
					return n, r.fail("missing line ending")
				}
			}
			end := nl
			if nl > 0 && r.buf[nl-1] == '\r' {
				nl--
			}
			if nl == 0 {
				r.buf = r.buf[end+1:]
				continue
			}
			if bytes.Equal(r.buf[:nl], tokReadEmptyLine) {
				r.buf = r.buf[end+1:]
				continue
			}
			trimmed := bytes.TrimRight(r.buf, "\r\n")
			if !bytes.Equal(trimmed, tokReadEnd) {
				return n, r.fail("unexpected end")
			}

			r.state = readDone

		case readDone:
			return n, io.EOF

		default:
			panic("unknown state")
		}
	}
}
