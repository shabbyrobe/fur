package gopher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	"github.com/shabbyrobe/fur/internal/uuencode"
)

var lineEnding = []byte{'\r', '\n'}

type ResponseClass int

const (
	BinaryClass ResponseClass = 1
	DirClass    ResponseClass = 2
	TextClass   ResponseClass = 3
	ErrorClass  ResponseClass = 4
)

type Response interface {
	URL() URL
	Class() ResponseClass
	Close() error
}

type BinaryResponse struct {
	url   URL
	inner io.ReadCloser
}

var _ Response = &BinaryResponse{}

func NewBinaryResponse(u URL, rdr io.ReadCloser) *BinaryResponse {
	return &BinaryResponse{url: u, inner: rdr}
}

func (br *BinaryResponse) URL() URL             { return br.url }
func (br *BinaryResponse) Close() error         { return br.inner.Close() }
func (br *BinaryResponse) Class() ResponseClass { return BinaryClass }

func (br *BinaryResponse) Read(b []byte) (n int, err error) {
	return br.inner.Read(b)
}

type UUEncodedResponse struct {
	url URL
	uu  *uuencode.Reader
	cls io.Closer
}

var _ Response = &UUEncodedResponse{}

func NewUUEncodedResponse(u URL, rdr io.ReadCloser) *UUEncodedResponse {
	dot := textproto.NewReader(bufio.NewReader(rdr)).DotReader()
	wrp := &expectedUnexpectedEOFReader{rdr: dot}
	uu := uuencode.NewReader(wrp, nil)
	return &UUEncodedResponse{url: u, uu: uu, cls: rdr}
}

func (br *UUEncodedResponse) File() (string, bool)      { return br.uu.File() }
func (br *UUEncodedResponse) Mode() (os.FileMode, bool) { return br.uu.Mode() }
func (br *UUEncodedResponse) Class() ResponseClass      { return BinaryClass }
func (br *UUEncodedResponse) URL() URL                  { return br.url }
func (br *UUEncodedResponse) Close() error              { return br.cls.Close() }

func (br *UUEncodedResponse) Read(b []byte) (n int, err error) {
	return br.uu.Read(b)
}

type TextResponse struct {
	rdr io.Reader
	url URL
	cls io.ReadCloser
}

var _ Response = &TextResponse{}

func NewTextResponse(u URL, rdr io.ReadCloser) *TextResponse {
	dot := textproto.NewReader(bufio.NewReader(rdr)).DotReader()
	wrp := &expectedUnexpectedEOFReader{rdr: dot}
	return &TextResponse{url: u, rdr: wrp, cls: rdr}
}

func (br *TextResponse) Class() ResponseClass { return TextClass }
func (br *TextResponse) URL() URL             { return br.url }
func (br *TextResponse) Close() error         { return br.cls.Close() }

func (br *TextResponse) Read(b []byte) (n int, err error) {
	return br.rdr.Read(b)
}

type Dirent struct {
	ItemType ItemType
	Display  string
	URL      URL
	Plus     bool

	Valid bool
	Error string
	Raw   string
}

type DirResponse struct {
	url    URL
	cls    io.Closer
	scn    *bufio.Scanner
	err    error
	line   int
	strict bool

	pos, n int
}

var _ Response = &DirResponse{}

func NewDirResponse(u URL, rdr io.ReadCloser) *DirResponse {
	dot := textproto.NewReader(bufio.NewReader(rdr)).DotReader()
	wrp := &expectedUnexpectedEOFReader{rdr: dot}
	scn := bufio.NewScanner(wrp)
	return &DirResponse{
		url: u,
		cls: rdr,
		scn: scn,
	}
}

func (br *DirResponse) Class() ResponseClass { return DirClass }
func (br *DirResponse) URL() URL             { return br.url }

func (br *DirResponse) Close() error {
	err := br.err
	if err == io.EOF {
		err = nil
	}
	if cerr := br.cls.Close(); err == nil && cerr != nil {
		err = cerr
	}
	return err
}

func (br *DirResponse) Next(dir *Dirent) bool {
	if br.err != nil {
		return false
	}

retry:
	if !br.scn.Scan() {
		br.err = br.scn.Err()
		return false
	}
	br.line++

	txt := br.scn.Text()
	tsz := len(txt)
	if tsz == 0 {
		goto retry
	}

	start := 1
	field := 0

	dir.URL = URL{}
	dir.ItemType = ItemType(txt[0])
	dir.URL.ItemType = ItemType(txt[0])
	dir.Valid = true
	dir.Raw = txt

	for i := start; i <= tsz; i++ {
		if i == tsz || txt[i] == '\t' {
			switch field {
			case 0:
				dir.Display = txt[start:i]
				field, start = field+1, i+1
			case 1:
				dir.URL.Selector = txt[start:i]
				field, start = field+1, i+1
			case 2:
				dir.URL.Hostname = txt[start:i]
				field, start = field+1, i+1

			case 3:
				// XXX: Things can get a bit fouled up by bad servers; telefisk.org serves mail
				// archives with bad whitespace in 'i' lines:
				// gopher://telefisk.org/1/mailarchives/gopher/gopher-2014-12.mbox%3F133
				//
				// If we can accept the server's output without doing something
				// unreasonable, we should try, so let's chop whitespace and skip empty
				// strings. Some hosts will fill these fields out with dummy data, so we
				// can't just presume that 'i' means concatenate all fields together and
				// presume that's the line; I think telefisk.org is just serving files up
				// as-is and prepending 'i' to every line regardless of whether that's
				// valid.
				ps := strings.TrimSpace(txt[start:i])
				if ps != "" {
					pi, err := strconv.ParseInt(ps, 10, 0)
					if err != nil {
						br.err = fmt.Errorf("gopher: unexpected port %q at line %d: %w", ps, br.line, err)
						return false
					}
					dir.URL.Port = int(pi)
				}
				field, start = field+1, i+1

			case 4:
				ps := txt[start:i]
				if ps == "+" {
					dir.Plus = true
				} else if ps != "" {
					br.err = fmt.Errorf("gopher: unexpected 'plus' field at line %d; expected '+' or '', found %q", br.line, ps)
					return false
				}

			case 5:
				br.err = fmt.Errorf("gopher: extra fields at line %d: %q", br.line, txt[start:i])
				return false
			}
		}
	}

	fieldLimit := 4
	if dir.ItemType == Info {
		// XXX: Lots of servers don't fill out the extra fields for 'i' lines:
		fieldLimit = 1
	}

	if field < fieldLimit {
		br.err = fmt.Errorf("gopher: missing fields at line %d: %q", br.line, txt)
		return false
	}

	return true
}

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
