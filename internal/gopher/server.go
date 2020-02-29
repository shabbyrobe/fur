package gopher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultRequestSizeLimit    = 1 << 12
	DefaultReadTimeout         = 10 * time.Second
	DefaultReadSelectorTimeout = 5 * time.Second
)

var (
	ErrBadRequest   = errors.New("gopher: bad request")
	ErrServerClosed = errors.New("gopher: server closed")
)

func ListenAndServe(addr string, host string, handler Handler) error {
	server := &Server{Handler: handler}
	return server.ListenAndServe(addr, host)
}

type Server struct {
	Handler     Handler
	MetaHandler MetaHandler
	ErrorLog    Logger
	Info        *ServerInfo

	RequestSizeLimit    int
	ReadTimeout         time.Duration
	ReadSelectorTimeout time.Duration

	conns     map[net.Conn]struct{}
	listeners map[net.Listener]struct{}
	lock      sync.Mutex
}

func (srv *Server) ListenAndServe(addr string, host string) error {
	if addr == "" {
		addr = ":gopher"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln, host)
}

func (srv *Server) Close() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	for l := range srv.listeners {
		l.Close()
	}
	for c := range srv.conns {
		c.Close()
	}

	return nil
}

func (srv *Server) metaHandler() MetaHandler {
	if srv.MetaHandler != nil {
		return srv.MetaHandler
	}
	mh, ok := srv.Handler.(MetaHandler)
	if ok {
		return mh
	}
	return nil
}

func (srv *Server) Serve(l net.Listener, host string) error {
	srv.addListener(l)

	var lhost, lport string
	var err error
	if host != "" {
		lhost, lport, err = resolveHostPort(host)
		if err != nil {
			return err
		}
	}

	var metaHandler = srv.metaHandler()
	var log = srv.ErrorLog
	if log == nil {
		log = stdLogger
	}

	var tempDelay time.Duration // http.Server trick for dealing with accept failure

	for {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if tempDelay > 1*time.Second {
					tempDelay = 1 * time.Second
				}
				log.Printf("gopher: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay) // XXX: can't be cancelled
				continue

			} else {
				return err
			}
		}

		tempDelay = 0
		chost, cport := lhost, lport
		if chost == "" {
			chost, cport, err = resolveHostPort(conn.LocalAddr().String())
			if err != nil {
				return err
			}
		}

		ctx := context.Background()
		buf := make([]byte, srv.requestSizeLimit())
		c := &serveConn{
			rwc: conn, srv: srv, buf: buf,
			host: chost, port: cport,
			log: log, meta: metaHandler,
		}
		srv.addConn(conn)
		go c.serve(ctx)
	}

	return nil
}

func (srv *Server) info() *ServerInfo {
	if srv.Info != nil {
		return srv.Info
	}
	var d = defaultServerInfo
	return &d
}

func (srv *Server) addListener(l net.Listener) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.listeners == nil {
		srv.listeners = make(map[net.Listener]struct{})
	}
	srv.listeners[l] = struct{}{}
}

func (srv *Server) removeListener(l net.Listener) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	delete(srv.listeners, l)
}

func (srv *Server) addConn(conn net.Conn) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.conns == nil {
		srv.conns = make(map[net.Conn]struct{})
	}
	srv.conns[conn] = struct{}{}
}

func (srv *Server) removeConn(conn net.Conn) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	delete(srv.conns, conn)
}

func (srv *Server) readTimeout() time.Duration {
	if srv.ReadTimeout != 0 {
		return srv.ReadTimeout
	}
	return DefaultReadTimeout
}

func (srv *Server) readSelectorTimeout() time.Duration {
	if srv.ReadSelectorTimeout != 0 {
		return srv.ReadSelectorTimeout
	}
	if srv.ReadTimeout != 0 {
		return srv.ReadTimeout
	}
	return DefaultReadSelectorTimeout
}

func (srv *Server) requestSizeLimit() int {
	requestLimit := srv.RequestSizeLimit
	if requestLimit <= 0 {
		requestLimit = DefaultRequestSizeLimit
	}
	return requestLimit
}

type serveConn struct {
	srv *Server
	rwc net.Conn
	buf []byte

	host string
	port string
	log  Logger
	meta MetaHandler
}

func (c *serveConn) serve(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			_, file, line, _ := runtime.Caller(2)
			remoteAddr := c.rwc.RemoteAddr().String()
			c.log.Printf("gopher: panic serving %s at %s:%d: %v\n", remoteAddr, file, line, err)
		}
	}()

	defer c.rwc.Close()
	defer c.srv.removeConn(c.rwc)

	c.rwc.SetReadDeadline(time.Now().Add(c.srv.readSelectorTimeout()))

	req, err := c.readRequest(ctx)
	if err != nil {
		// FIXME: log
		c.rwc.Close()
		return
	}

	if req.url.IsMeta() && c.meta != nil {
		mw := newMetaWriter(c.rwc, req)
		c.meta.ServeGopherMeta(ctx, mw, req)
		if !mw.flushed {
			if err := mw.Flush(); err != nil {
				panic(err)
			}
		}

	} else {
		c.srv.Handler.ServeGopher(ctx, c.rwc, req)
	}
}

func (c *serveConn) readRequest(ctx context.Context) (req *Request, err error) {
	var nl, at, sz int
	for {
		n, err := c.rwc.Read(c.buf[sz:])
		if err != nil && (err != io.EOF || n == 0) {
			return nil, err
		}
		sz += n

		for i := at; i < sz; i++ {
			if c.buf[i] == '\n' {
				nl = i
				goto found
			}
		}
	}

found:
	line, left := c.buf[:nl], c.buf[nl+1:]
	line = dropCR(line)
	sz = len(line)

	var selector, view string
	var hasData bool
	var field, s int

	for i := at; i < sz; i++ {
		if i == sz || line[i] == '\t' {
			switch field {
			case 0:
				selector = string(c.buf[s:i])
				field, s = field+1, i+1

			case 1:
				view = string(c.buf[s:i])
				field, s = field+1, i+1

			case 2:
				ok := i-s == 1 && (c.buf[s] == '0' || c.buf[s] == '1')
				if !ok {
					// FIXME: respond with error?
					return nil, ErrBadRequest
				}
				hasData = c.buf[s] == '1'
				field, s = field+1, i+1

			case 3:
				// XXX: Gopher clients could send us any old garbage. Let's just
				// ignore it for now.
				return nil, ErrBadRequest
			}
		}
	}

	var body io.ReadCloser = c.rwc
	if len(left) > 0 || hasData {
		multi := io.MultiReader(bytes.NewReader(left), c.rwc)
		body = &readCloser{
			readFn:  multi.Read,
			closeFn: c.rwc.Close,
		}
	}

	url := URL{
		Hostname: c.host,
		Port:     c.port,
		Root:     true,
		ItemType: 0,
		Selector: selector,
		Search:   view,
	}

	rq := NewRequest(url, body)
	rq.RemoteAddr = c.rwc.RemoteAddr().(*net.TCPAddr)

	return rq, nil
}

func resolveHostPort(host string) (rhost string, rport string, err error) {
	rhost, rport, err = net.SplitHostPort(host)
	if err != nil {
		// SplitHostPort has uncatchable errors, so let's just be brutes about it:
		var retryErr error
		rhost, rport, retryErr = net.SplitHostPort(host + ":70")
		if retryErr != nil {
			return rhost, rport, err // return orig error
		}
	}

	return rhost, rport, nil
}
