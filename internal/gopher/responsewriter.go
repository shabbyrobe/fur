package gopher

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
)

type ResponseWriter interface {
	Write([]byte) (int, error)
}

func NotFound(w ResponseWriter, r *Request) {
	dw := NewDirWriter(w, r)
	defer MustFlush(dw)
	dw.Error(fmt.Sprintf("Not found: %s", r.URL()))
}

type TextWriter struct {
	bufw *bufio.Writer
	last byte
}

func NewTextWriter(w io.Writer) *TextWriter {
	return &TextWriter{
		bufw: bufio.NewWriter(w),
	}
}

func (tw *TextWriter) WriteString(s string) (n int, err error) {
	return tw.Write([]byte(s))
}

func (tw *TextWriter) WriteLine(s string) (n int, err error) {
	wn, _ := tw.Write([]byte(s))
	n += wn
	wn, err = tw.Write(crlf)
	n += wn
	return n, err
}

func (tw *TextWriter) Flush() error {
	if tw.last != '\n' {
		tw.bufw.Write(crlf)
	}
	tw.bufw.Write(dotTerminator)
	return tw.bufw.Flush()
}

func (tw *TextWriter) MustFlush() { MustFlush(tw) }

func (tw *TextWriter) Write(b []byte) (n int, err error) {
	blen := len(b)
	s := 0
	for i := 0; i < blen; i++ {
		if b[i] == '\n' && tw.last != '\r' {
			tw.bufw.Write(crlf)
			if _, err := tw.bufw.Write(b[s:i]); err != nil {
				return s, err
			}
			s = i + 1
		}
		tw.last = b[i]
	}
	if _, err := tw.bufw.Write(b[s:]); err != nil {
		return blen, err
	}
	return blen, nil
}

type DirWriter struct {
	bufw *bufio.Writer
	host string
	port string
}

func NewDirWriter(w io.Writer, rq *Request) *DirWriter {
	u := rq.URL()
	port, err := net.LookupPort("tcp", u.Port)
	if err != nil {
		panic(err)
	}
	return &DirWriter{
		bufw: bufio.NewWriter(w),
		host: u.Hostname,
		port: strconv.FormatInt(int64(port), 10),
	}
}

func (dw *DirWriter) Dirent(dirent *Dirent) error {
	return dirent.write(dw.bufw)
}

// Info writes an 'i' line to the directory. It is safe to ignore the returned
// error as it will be returned when Flush() is called.
func (dw *DirWriter) Info(disp string) error {
	bufw := dw.bufw
	bufw.WriteByte(Info)
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.Write(tokNull)
	bufw.WriteByte('\t')
	bufw.Write(tokInvalid)
	bufw.WriteByte('\t')
	bufw.WriteByte('0')
	_, err := bufw.Write(crlf)
	return err
}

func (dw *DirWriter) RemoteSelector(i ItemType, disp, sel string, host string, port int) error {
	bufw := dw.bufw
	bufw.WriteByte(byte(i))
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.WriteString(sel)
	bufw.WriteByte('\t')
	bufw.WriteString(host)
	bufw.WriteByte('\t')
	bufw.WriteString(strconv.FormatInt(int64(port), 10))
	_, err := bufw.Write(crlf)
	return err
}

func (dw *DirWriter) Text(disp, sel string) error {
	return dw.Selector(Text, disp, sel)
}

func (dw *DirWriter) Root(disp string) error {
	return dw.Selector(Dir, disp, "")
}

func (dw *DirWriter) Dir(disp, sel string) error {
	return dw.Selector(Dir, disp, sel)
}

func (dw *DirWriter) Binary(disp, sel string) error {
	return dw.Selector(Binary, disp, sel)
}

func (dw *DirWriter) Image(disp, sel string) error {
	return dw.Selector(Image, disp, sel)
}

func (dw *DirWriter) Selector(i ItemType, disp, sel string) error {
	bufw := dw.bufw
	bufw.WriteByte(byte(i))
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.WriteString(sel)
	bufw.WriteByte('\t')
	bufw.WriteString(dw.host)
	bufw.WriteByte('\t')
	bufw.WriteString(dw.port)
	_, err := bufw.Write(crlf)
	return err
}

func (dw *DirWriter) Plus(i ItemType, disp, sel string) error {
	bufw := dw.bufw
	bufw.WriteByte(byte(i))
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.WriteString(sel)
	bufw.WriteByte('\t')
	bufw.WriteString(dw.host)
	bufw.WriteByte('\t')
	bufw.WriteString(dw.port)
	bufw.WriteByte('\t')
	bufw.WriteByte('+')
	_, err := bufw.Write(crlf)
	return err
}

func (dw *DirWriter) Error(disp string) error {
	bufw := dw.bufw
	bufw.WriteByte(byte(ItemError))
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.Write(tokNull)
	bufw.WriteByte('\t')
	bufw.Write(tokInvalid)
	bufw.WriteByte('\t')
	bufw.WriteByte('0')
	_, err := bufw.Write(crlf)
	return err
}

func (dw *DirWriter) Flush() error {
	dw.bufw.Write(dotTerminator)
	return dw.bufw.Flush()
}

func (dw *DirWriter) MustFlush() { MustFlush(dw) }

var (
	crlf          = []byte{'\r', '\n'}
	dotTerminator = []byte{'.', '\r', '\n'}
	tokInvalid    = []byte("invalid")
	tokNull       = []byte("null")
)
