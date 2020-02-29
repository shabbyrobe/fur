package gopher

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Client struct {
	Timeout               time.Duration
	ExtraBinaryTypes      [256]bool
	DisableErrorIntercept bool // Warning: subject to change.

	Recorder        Recorder
	CapsSource      CapsSource
	TLSClientConfig *tls.Config
	TLSMode         TLSMode
}

func (c *Client) timeoutDial() time.Duration {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return timeout
}

// FIXME: timeouts are currently shared with dial; should be separated
func (c *Client) timeoutRead() time.Duration  { return c.timeoutDial() }
func (c *Client) timeoutWrite() time.Duration { return c.timeoutDial() }

func (c *Client) dial(ctx context.Context, rq *Request, tlsMode TLSMode) (net.Conn, error) {
	if !rq.url.CanFetch() {
		return nil, fmt.Errorf("gopher: cannot fetch URL %q", rq.url)
	}

	dialer := net.Dialer{Timeout: c.timeoutDial()}
	conn, err := dialer.DialContext(ctx, "tcp", rq.url.Host())
	if err != nil {
		return nil, err
	}

	if tlsMode.shouldAttempt() {
		var tlsConf *tls.Config
		if c.TLSClientConfig == nil {
			tlsConf = &tls.Config{}
		} else {
			tlsConf = c.TLSClientConfig.Clone()
		}

		tlsConf.ServerName = rq.url.Hostname
		conn = tls.Client(conn, tlsConf)
	}

	return conn, nil
}

// send the request for URL u to conn. A non-nil response is returned if the response is
// intercepted (i.e. in the case of error), otherwise the caller should use conn to read
// the repsonse.
//
// Callers must use the reader returned by this function rather than the conn to read
// the response.
func (c *Client) send(ctx context.Context, conn net.Conn, rq *Request, at time.Time, interceptErrors bool) (net.Conn, error) {
	var rec Recording

	caps, err := c.loadCaps(ctx, rq.url.Hostname, rq.url.Port)
	if err != nil {
		return conn, err
	}
	_ = caps // XXX: not using yet.

	if c.Recorder != nil {
		rec = c.Recorder.BeginRecording(rq, at)
		conn = recordConn(rec, conn)
	}

	if err := conn.SetWriteDeadline(at.Add(c.timeoutWrite())); err != nil {
		return conn, err
	}

	var buf bytes.Buffer
	if err := rq.buildSelector(&buf); err != nil {
		return conn, fmt.Errorf("gopher: failed to build selector: %w", err)
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		// XXX: We must make sure to return this error as-is so we can catch and retry in
		// dialAndSend. We avoid errors.As because it introduces a bucketload of slow.
		if tlserr, ok := err.(tls.RecordHeaderError); ok {
			return conn, tlserr
		}
		return conn, fmt.Errorf("gopher: request selector write error: %w", err)
	}

	if body := rq.Body(); body != nil {
		if _, err := io.Copy(conn, body); err != nil {
			return conn, err
		}
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
			return NewError(rq.url, status, msg, confidence)
		})
		if rsErr != nil {
			rsErr.Raw = scratch
			return conn, rsErr
		}
		conn = &bufferedConn{conn, io.MultiReader(bytes.NewReader(scratch), conn)}
	}

	return conn, nil
}

func (c *Client) loadCaps(ctx context.Context, host string, port string) (caps Caps, err error) {
	if c.CapsSource != nil {
		caps, err = c.CapsSource.LoadCaps(ctx, host, port)
		if err != nil {
			return nil, err
		}
	}
	if caps == nil {
		caps = DefaultCaps
	}
	return caps, nil
}

func (c *Client) dialAndSend(ctx context.Context, rq *Request, at time.Time, interceptErrors bool) (net.Conn, error) {
	tlsMode := c.TLSMode.resolve(rq.url.Secure)
	conn, err := c.dial(ctx, rq, tlsMode)
	if err != nil {
		return nil, err
	}

retry:
	rdr, err := c.send(ctx, conn, rq, at, interceptErrors)
	if err != nil {
		conn.Close()

		if _, ok := err.(tls.RecordHeaderError); ok && tlsMode.downgrade() {
			tlsMode = TLSDisabled
			conn, err = c.dial(ctx, rq, tlsMode)
			if err == nil {
				goto retry
			}
		}
		return nil, err
	}

	return rdr, nil
}

func (c *Client) Fetch(ctx context.Context, rq *Request) (Response, error) {
	it := rq.url.ItemType
	if rq.url.Root {
		it = Dir
	}

	// FIXME: meta response

	if it.IsBinary() || c.ExtraBinaryTypes[it] {
		return c.Binary(ctx, rq)
	}
	switch it {
	case UUEncoded:
		return c.UUEncoded(ctx, rq)
	case Dir, Search:
		return c.Dir(ctx, rq)
	}
	return c.Text(ctx, rq)
}

func (c *Client) Search(ctx context.Context, rq *Request) (*DirResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewDirResponse(rq, conn), nil
}

func (c *Client) Dir(ctx context.Context, rq *Request) (*DirResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, fmt.Errorf("gopher: dir request failed: %w", err)
	}
	return NewDirResponse(rq, conn), nil
}

func (c *Client) Text(ctx context.Context, rq *Request) (*TextResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewTextResponse(rq, conn), nil
}

func (c *Client) Binary(ctx context.Context, rq *Request) (*BinaryResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewBinaryResponse(rq, conn), nil
}

func (c *Client) UUEncoded(ctx context.Context, rq *Request) (*UUEncodedResponse, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, !c.DisableErrorIntercept)
	if err != nil {
		return nil, err
	}
	return NewUUEncodedResponse(rq, conn), nil
}

func (c *Client) Raw(ctx context.Context, rq *Request) (Response, error) {
	start := time.Now()
	conn, err := c.dialAndSend(ctx, rq, start, false)
	if err != nil {
		return nil, err
	}
	return NewBinaryResponse(rq, conn), nil
}
