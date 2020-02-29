package gopher

import (
	"bufio"
	"io"
	"os"

	"github.com/shabbyrobe/fur/internal/uuencode"
)

var lineEnding = []byte{'\r', '\n'}

type Response interface {
	Reader() io.ReadCloser
	Status() Status
	Request() *Request
	Class() Class
	Close() error
}

type BinaryResponse struct {
	rq    *Request
	inner io.ReadCloser
}

var _ Response = &BinaryResponse{}

func NewBinaryResponse(rq *Request, rdr io.ReadCloser) *BinaryResponse {
	return &BinaryResponse{rq: rq, inner: rdr}
}

func (br *BinaryResponse) Request() *Request     { return br.rq }
func (br *BinaryResponse) Class() Class          { return BinaryClass }
func (br *BinaryResponse) Status() Status        { return OK }
func (br *BinaryResponse) Reader() io.ReadCloser { return br }
func (br *BinaryResponse) Close() error          { return br.inner.Close() }

func (br *BinaryResponse) Read(b []byte) (n int, err error) {
	return br.inner.Read(b)
}

type UUEncodedResponse struct {
	rq  *Request
	uu  *uuencode.Reader
	cls io.Closer
}

var _ Response = &UUEncodedResponse{}

func NewUUEncodedResponse(rq *Request, rdr io.ReadCloser) *UUEncodedResponse {
	uu := uuencode.NewReader(NewTextReader(rdr), nil)
	return &UUEncodedResponse{rq: rq, uu: uu, cls: rdr}
}

func (br *UUEncodedResponse) File() (string, bool)      { return br.uu.File() }
func (br *UUEncodedResponse) Mode() (os.FileMode, bool) { return br.uu.Mode() }

func (br *UUEncodedResponse) Request() *Request     { return br.rq }
func (br *UUEncodedResponse) Class() Class          { return BinaryClass }
func (br *UUEncodedResponse) Reader() io.ReadCloser { return br }
func (br *UUEncodedResponse) Close() error          { return br.cls.Close() }
func (br *UUEncodedResponse) Status() Status        { return OK }

func (br *UUEncodedResponse) Read(b []byte) (n int, err error) {
	return br.uu.Read(b)
}

type TextResponse struct {
	rdr io.Reader
	rq  *Request
	cls io.ReadCloser
}

var _ Response = &TextResponse{}

func NewTextResponse(rq *Request, rdr io.ReadCloser) *TextResponse {
	return &TextResponse{rq: rq, rdr: NewTextReader(rdr), cls: rdr}
}

func (br *TextResponse) Class() Class          { return TextClass }
func (br *TextResponse) Request() *Request     { return br.rq }
func (br *TextResponse) Status() Status        { return OK }
func (br *TextResponse) Reader() io.ReadCloser { return br }
func (br *TextResponse) Close() error          { return br.cls.Close() }

func (br *TextResponse) Read(b []byte) (n int, err error) {
	return br.rdr.Read(b)
}

type DirResponse struct {
	rq   *Request
	cls  io.Closer
	scn  *bufio.Scanner
	rdr  io.Reader
	err  error
	line int

	pos, n int
}

var _ Response = &DirResponse{}

func NewDirResponse(rq *Request, rdr io.ReadCloser) *DirResponse {
	dot := NewTextReader(rdr)
	scn := bufio.NewScanner(dot)
	return &DirResponse{
		rq:  rq,
		cls: rdr,
		scn: scn,
		rdr: dot,
	}
}

func (br *DirResponse) Status() Status    { return OK }
func (br *DirResponse) Class() Class      { return DirClass }
func (br *DirResponse) Request() *Request { return br.rq }

func (br *DirResponse) Reader() io.ReadCloser {
	return &readCloser{
		readFn:  br.rdr.Read,
		closeFn: br.Close,
	}
}

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
	if len(txt) == 0 {
		goto retry
	}

	if err := parseDirent(txt, br.line, dir); err != nil {
		br.err = err
		return false
	}

	return true
}
