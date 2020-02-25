package gopher

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
)

type Request struct {
	url  URL
	body io.ReadCloser

	View string

	// When a server accepts an actual connection, this will be set to the remote address.
	// This field is ignored by the Gopher client.
	RemoteAddr *net.TCPAddr
}

func NewRequest(url URL, body io.Reader) *Request {
	var ok bool
	var rc io.ReadCloser
	if body == nil {
		rc = nilReadCloserVal
	} else {
		rc, ok = body.(io.ReadCloser)
		if !ok {
			rc = ioutil.NopCloser(body)
		}
	}
	return &Request{
		url:  url,
		body: rc,
	}
}

func (r *Request) URL() URL {
	return r.url
}

func (r *Request) Body() io.ReadCloser {
	return r.body
}

func (r *Request) buildSelector(buf *bytes.Buffer) error {
	buf.WriteString(r.url.Selector)
	buf.WriteByte('\t')
	buf.WriteString(r.url.Search)

	buf.WriteByte('\t')
	buf.WriteString(r.View)

	if r.body != nil {
		buf.WriteByte('1')
	} else {
		buf.WriteByte('0')
	}

	buf.WriteString("\r\n")
	return nil
}
