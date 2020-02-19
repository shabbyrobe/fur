package gopher

import (
	"context"
	"fmt"
	"net"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Client struct {
	Timeout     time.Duration
	BinaryTypes [256]bool
}

func (c *Client) timeout() time.Duration {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return timeout
}

func (c *Client) send(ctx context.Context, u URL, at time.Time) (*net.TCPConn, error) {
	if !u.ItemType.CanFetch() {
		return nil, fmt.Errorf("gopher: cannot fetch URL %q", u)
	}

	timeout := c.timeout()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", u.Host())
	if err != nil {
		return nil, err
	}

	tcp := conn.(*net.TCPConn)
	if err := conn.SetWriteDeadline(at.Add(timeout)); err != nil {
		tcp.Close()
		return nil, err
	}
	if _, err := conn.Write([]byte(u.Query())); err != nil {
		tcp.Close()
		return nil, err
	}

	return tcp, nil
}

func (c *Client) sendWithReadTimeout(ctx context.Context, u URL, at time.Time) (*net.TCPConn, error) {
	tcp, err := c.send(ctx, u, at)
	if err != nil {
		return nil, err
	}
	if err := tcp.SetReadDeadline(at.Add(c.timeout())); err != nil {
		tcp.Close()
		return nil, err
	}
	return tcp, err
}

// func (c *Client) interceptError(ctx context.Context, tcp *net.TCPConn) (rdr io.ReadCloser, rsErr *ErrorResponse, err error) {
//     first := make([]byte, 2048)
//
//     // XXX: this is difficult... we can only try to Read() once because subsequent calls
//     // to Read() may block, which we can't allow because we have no way to know when
//     // to unblock. Unfortunately, the server could be written to write bytes 1 at a
//     // time (it probably won't, but if it does, we're stuffed), or to write the whole
//     // response in one hit. The network could chop the reads up to the MTU size. All
//     // sorts of stuff.
//     //
//     // So detecting the error can only be done with the result of the first call to Read.
//     n, err := tcp.Read(first)
//     if err != nil {
//         tcp.Close()
//         return nil, nil, err
//     }
//
//     first = first[:n]
//
//     // Step 0: Empty file == error?
//
//     // Step 1: Try to detect error responses that start with '--':
//     // https://tools.ietf.org/html/draft-matavka-gopher-ii-02#section-9
//
//     // Step 2: Try to detect error responses that are a single directory entry of type
//     // '3', which may be preceded and/or followed by a list of 'i' lines.
//
//     // Step 3: Iterate well known server responses for errors
//     // - 3Happy helping ☃ here: Sorry, your selector does not start with / or contains '..'. That's illegal here.	Err	localhost	70
//     // - An error occurred: Resource not found.
//     // - File: '...' not found.
//     // - Error: resource caps.txt does not exist on ...
//     // - Error: 404 Not Found
//     // - Error: File or directory not found!
//
//     return tcp, nil, nil
// }
//
func (c *Client) Fetch(ctx context.Context, u URL) (Response, error) {
	it := u.ItemType
	if u.Root {
		it = Dir
	}
	if it.IsBinary() || c.BinaryTypes[it] {
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
	tcp, err := c.sendWithReadTimeout(ctx, u, start)
	if err != nil {
		return nil, err
	}
	return NewDirResponse(u, tcp), nil
}

func (c *Client) Dir(ctx context.Context, u URL) (*DirResponse, error) {
	start := time.Now()
	tcp, err := c.sendWithReadTimeout(ctx, u, start)
	if err != nil {
		return nil, err
	}
	return NewDirResponse(u, tcp), nil
}

func (c *Client) Text(ctx context.Context, u URL) (*TextResponse, error) {
	start := time.Now()
	tcp, err := c.sendWithReadTimeout(ctx, u, start)
	if err != nil {
		return nil, err
	}
	return NewTextResponse(u, tcp), nil
}

func (c *Client) Binary(ctx context.Context, u URL) (*BinaryResponse, error) {
	start := time.Now()
	tcp, err := c.sendWithReadTimeout(ctx, u, start)
	if err != nil {
		return nil, err
	}
	return NewBinaryResponse(u, tcp), nil
}

func (c *Client) UUEncoded(ctx context.Context, u URL) (*UUEncodedResponse, error) {
	start := time.Now()
	tcp, err := c.sendWithReadTimeout(ctx, u, start)
	if err != nil {
		return nil, err
	}
	return NewUUEncodedResponse(u, tcp), nil
}
