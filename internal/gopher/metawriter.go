package gopher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrMetaLeadingPlus     = errors.New("gopher: metadata value contains leading '+'")
	ErrMetaInfoAlreadySent = errors.New("gopher: attempted to send INFO more than once for '!' meta request")
	ErrMetaInfoNotSent     = errors.New("gopher: attempted to send non-info record before INFO")

	errMetaAlreadyBegan   = errors.New("gopher: metadata already began")
	errMetaInfoAfterError = errors.New("gopher: meta INFO record sent after ERROR")

	tokMetaTextBegin = []byte("+-1") // gopher-ii-03, 6
	tokMetaInfo      = []byte("+INFO: ")
)

type MetaWriter interface {
	// The `INFO` record is MANDATORY in every metadata listing.  It
	// contains the same data as the Gopher selector, with a plus sign at
	// the end, per GopherIIbis style. It MUST always be present, and it
	// MUST always be the first metadata record present. The `INFO` record
	// serves to separate metadata listings when more are sent at the same
	// time.
	//
	// If the Request is for a single file's metadata only ('!' rather than
	// '&'), multiple calls to Info() will result in an error.
	Info(i ItemType, disp, sel string)

	// If the Request is invalid, MetaError() should be called once and only
	// once. No other calls to write Info or Records are valid if MetaError()
	// has been called.
	MetaError(code Status, msgfmt string)

	// Write an entire record with a string value:
	WriteRecord(record string, value string) (ok bool)

	// Begin a record and return a writer that can be used to write its value.
	// If the returned writer is nil, the record is ignored by the request.
	// Duplicate records may be written.
	BeginRecord(record string) *MetaValueWriter

	// Flush any buffered metadata and return any cached error. It is not
	// necessary to call Flush() directly; Server will call it regardless
	// at the end of the request.
	Flush() error
}

type MetaEntry struct {
	Record string
	Value  string
}

func WriteMeta(mw MetaWriter, i ItemType, disp, sel string, meta []MetaEntry) error {
	mw.Info(i, disp, sel)
	for _, e := range meta {
		mw.WriteRecord(e.Record, e.Value)
	}
	return mw.Flush()
}

type metaWriter struct {
	bufw       *bufio.Writer
	rq         *Request
	began      bool
	infoSet    bool
	lastRecord *MetaValueWriter
	flushed    bool
	flushErr   error
	recordNum  int
}

var _ MetaWriter = &metaWriter{}

func newMetaWriter(w io.Writer, rq *Request) *metaWriter {
	if !rq.url.IsMeta() {
		// XXX: this may not be necessary but it was confounding some tests
		panic(fmt.Errorf("gopher: tried to write meta value for non-meta request"))
	}

	bufw := bufio.NewWriter(w)
	return &metaWriter{bufw: bufw, rq: rq}
}

func (mw *metaWriter) nextRecord(last bool) {
	if !last {
		mw.bufw.Write(crlf)
	}

	if mw.lastRecord != nil {
		if mw.lastRecord.err != nil {
			panic(mw.lastRecord.err)
		}

		if mw.lastRecord.last != '\n' {
			// FIXME: this doesn't collapse quite enough vertical whitespace. should
			// track two crlfs, not just one, to avoid the second crlf.
			mw.bufw.Write(crlf)
		}
	}

	mw.lastRecord = nil
	mw.recordNum++
}

func (mw *metaWriter) Flush() error {
	if mw.flushed {
		return mw.flushErr
	}
	if !mw.began {
		mw.beginMeta()
	}

	mw.nextRecord(true)
	mw.bufw.WriteByte('.')
	mw.bufw.Write(crlf)

	mw.flushErr = mw.bufw.Flush()
	mw.flushed = true

	return mw.flushErr
}

func (mw *metaWriter) beginMeta() {
	if mw.began {
		panic(errMetaAlreadyBegan)
	}
	mw.began = true
	mw.bufw.Write(tokMetaTextBegin)
}

func (mw *metaWriter) Info(i ItemType, disp, sel string) {
	if mw.infoSet && mw.rq.url.MetaType() == MetaItem {
		panic(ErrMetaInfoAlreadySent)
	}
	if !mw.began {
		mw.beginMeta()
	}

	mw.nextRecord(false) // XXX: do not move after 'infoSet = true'
	mw.infoSet = true

	bufw := mw.bufw
	bufw.Write(tokMetaInfo)

	bufw.WriteByte(byte(i))
	bufw.WriteString(disp)
	bufw.WriteByte('\t')
	bufw.WriteString(sel)
	bufw.WriteByte('\t')
	bufw.WriteString(mw.rq.url.Hostname)
	bufw.WriteByte('\t')
	bufw.WriteString(mw.rq.url.Port)
	bufw.WriteByte('\t')
	bufw.WriteByte('+')
	bufw.Write(crlf)

	// The sooner we flush the info line, the sooner clients can process it:
	if err := bufw.Flush(); err != nil {
		panic(err)
	}
}

func (mw *metaWriter) WriteRecord(record string, value string) (ok bool) {
	if mvw := mw.BeginRecord(record); mvw != nil {
		mvw.WriteString(value)
		return true
	}
	return false
}

func (mw *metaWriter) BeginRecord(record string) *MetaValueWriter {
	bufw := mw.bufw

	if record == "INFO" {
		mw.nextRecord(false)
		mw.infoSet = true
		bufw.Write(tokMetaInfo)

	} else {
		if !mw.infoSet {
			panic(ErrMetaInfoNotSent)
		}
		if !metaIncludesRecord(mw.rq.url.Search, record) {
			return nil
		}

		mw.nextRecord(false)
		bufw.WriteByte('+')
		bufw.WriteString(record)
		bufw.WriteByte(':')
		bufw.Write(crlf)
	}

	mw.lastRecord = &MetaValueWriter{bufw: bufw}
	return mw.lastRecord
}

// An example of a GopherIIbis error follows:
//	--404[CR][LF]The file requested could not be found.[CR][LF].[CR][LF]
//
func (mw *metaWriter) MetaError(code Status, msg string) {
	if mw.infoSet {
		panic(ErrMetaInfoAlreadySent)
	}
	if strings.IndexByte(msg, '\n') >= 0 {
		panic(fmt.Errorf("gopher: meta error message contained newlines"))
	}

	bufw := mw.bufw
	bufw.WriteByte(MetaError)
	bufw.WriteString(strconv.FormatInt(int64(code), 10))
	bufw.Write(crlf)

	bufw.WriteString(msg)
	bufw.Write(crlf)
}

type MetaValueWriter struct {
	bufw *bufio.Writer
	last byte
	col  int
	err  error
}

func (mvw *MetaValueWriter) WriteString(s string) (n int, err error) {
	return mvw.Write([]byte(s))
}

func (mvw *MetaValueWriter) WriteLine(s string) (n int, err error) {
	wn, _ := mvw.Write([]byte(s))
	n += wn
	wn, err = mvw.Write(crlf)
	n += wn
	return n, err
}

func (mvw *MetaValueWriter) Write(b []byte) (n int, err error) {
	if mvw.err != nil {
		return n, mvw.err
	}

	blen := len(b)
	s := 0
	for i := 0; i < blen; i++ {
		if b[i] == '\n' && mvw.last != '\r' {
			if _, err := mvw.bufw.Write(b[s:i]); err != nil {
				return s, err
			}
			mvw.bufw.Write(crlf)
			mvw.col = 0
			s = i + 1
		} else {
			mvw.col++
		}

		if mvw.col == 0 && b[i] == '+' {
			mvw.err = ErrMetaLeadingPlus
			return n, mvw.err
		}

		mvw.last = b[i]
	}

	wn, err := mvw.bufw.Write(b[s:])
	if err != nil {
		return blen, err
	}
	mvw.col += wn

	return blen, nil
}

func metaIncludesRecord(search string, record string) bool {
	slen := len(search)
	if slen == 0 {
		return false
	}
	if search[0] != '!' && search[0] != '&' {
		return false
	}
	if slen == 1 {
		return true
	}
	matched := 0
	rlen := len(record)
	for i := 1; i < slen; i++ {
		c := search[i]
		if c == '+' {
			if matched == rlen {
				return true
			}
			matched = 0
		}
		if c == record[matched] {
			matched++
		}
	}
	return matched == rlen
}
