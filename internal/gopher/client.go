package gopher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Client struct {
	Timeout          time.Duration
	ExtraBinaryTypes [256]bool

	// Disables error interception. Warning: subject to change.
	DisableErrorIntercept bool

	Recorder Recorder
}

func (c *Client) timeoutDial() time.Duration {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return timeout
}

// FIXME: separate timeouts
func (c *Client) timeoutRead() time.Duration  { return c.timeoutDial() }
func (c *Client) timeoutWrite() time.Duration { return c.timeoutDial() }

func (c *Client) dial(ctx context.Context, u URL) (*net.TCPConn, error) {
	if !u.CanFetch() {
		return nil, fmt.Errorf("gopher: cannot fetch URL %q", u)
	}

	dialer := net.Dialer{Timeout: c.timeoutDial()}
	conn, err := dialer.DialContext(ctx, "tcp", u.Host())
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

// send the request for URL u to conn. A non-nil response is returned if the response is
// intercepted (i.e. in the case of error), otherwise the caller should use conn to read
// the repsonse.
//
// Callers must use the reader returned by this function rather than the conn to read
// the response.
func (c *Client) send(ctx context.Context, conn conn, u URL, at time.Time, interceptErrors bool) (conn, error) {
	var rec Recording

	if c.Recorder != nil {
		rec = c.Recorder.BeginRecording(u, at)
		conn = recordConn(rec, conn)
	}

	if err := conn.SetWriteDeadline(at.Add(c.timeoutWrite())); err != nil {
		return conn, err
	}
	if _, err := conn.Write([]byte(u.Query())); err != nil {
		return conn, err
	}
	if err := conn.SetReadDeadline(at.Add(c.timeoutRead())); err != nil {
		return conn, err
	}

	if interceptErrors {
		// If the error isn't present in this, we can't detect it:
		const maxErrorRead = 1024

		scratch := make([]byte, maxErrorRead)

		// XXX: this is difficult... we can only try to Read() once because subsequent calls
		// to Read() may block, which we can't allow because we have no way to know when
		// to unblock. Unfortunately, the server could be written to write bytes 1 at a
		// time (it probably won't, but if it does, we're stuffed), or the network could
		// chop the reads up to some crazy MTU size (I've seen this go haywire with a
		// certain VPN client before). All sorts of stuff.
		//
		// XXX: update... bucktooth issues writes to the socket one dirent at a time
		// (which means we can't rely on being able to skip 'i' lines to get to the first
		// '3' line from a single read), so we will have to find a way to "read at least",
		// taking connection closes _and_ '.\r\n' into account to know when to stop.
		n, err := conn.Read(scratch)
		if n > 0 && err == io.EOF {
			err = nil
		}
		if err != nil {
			return conn, err
		}

		scratch = scratch[:n]
		rsErr := DetectError(scratch, func(status Status, msg string, confidence float64) *Error {
			if rec != nil {
				rec.SetStatus(status, msg)
			}
			return NewError(u, status, msg, confidence)
		})
		if rsErr != nil {
			rsErr.Raw = scratch
			return conn, rsErr
		}
		conn = &bufferedConn{conn, io.MultiReader(bytes.NewReader(scratch), conn)}
	}

	return conn, nil
}

func (c *Client) dialAndSend(ctx context.Context, u URL, at time.Time, interceptErrors bool) (conn, error) {
	conn, err := c.dial(ctx, u)
	if err != nil {
		return nil, err
	}

	rdr, err := c.send(ctx, conn, u, at, interceptErrors)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return rdr, nil
}

func (c *Client) Fetch(ctx context.Context, u URL) (Response, error) {
	it := u.ItemType
	if u.Root {
		it = Dir
	}
	if it.IsBinary() || c.ExtraBinaryTypes[it] {
		return c.Binary(ctx, u)
	}
	switch it {
	case UUEncoded:
		return c.UUEncoded(ctx, u)
	case Dir, Search:
		return c.Dir(ctx, u)
	}
	return c.Text(ctx, u)
}

func (c *Client) Search(ctx context.Context, u URL) (*DirResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewDirResponse(u, conn), nil
}

func (c *Client) Dir(ctx context.Context, u URL) (*DirResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewDirResponse(u, conn), nil
}

func (c *Client) Text(ctx context.Context, u URL) (*TextResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewTextResponse(u, conn), nil
}

func (c *Client) Binary(ctx context.Context, u URL) (*BinaryResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewBinaryResponse(u, conn), nil
}

func (c *Client) UUEncoded(ctx context.Context, u URL) (*UUEncodedResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewUUEncodedResponse(u, conn), nil
}

func (c *Client) Raw(ctx context.Context, u URL) (Response, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, u, start, false)
	if err != nil {
		return nil, err
	}
	return NewBinaryResponse(u, conn), nil
}

type conn interface {
	io.Reader
	io.Writer
	io.Closer

	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type bufferedConn struct {
	conn
	rdr io.Reader
}

func (bc *bufferedConn) Read(b []byte) (n int, err error) {
	return bc.rdr.Read(b)
}
