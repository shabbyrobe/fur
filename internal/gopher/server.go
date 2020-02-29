package gopher

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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
	ErrBadRequest             = errors.New("gopher: bad request")
	ErrServerClosed           = errors.New("gopher: server closed")
	ErrRequestFileFlagInvalid = errors.New("gopher: client sent an invalid file flag (gIIs6)")
	ErrRequestTrailingData    = errors.New("gopher: request contained invalid trailing data")
)

var (
	upgradeTLSErrorResponse = []byte("3Error\t\tinvalid\t0\r\n")
)

func ListenAndServe(addr string, host string, handler Handler, meta MetaHandler) error {
	server := &Server{Handler: handler, MetaHandler: meta}
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
	TLSConfig           *tls.Config

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
	srv   *Server
	rwc   net.Conn
	buf   []byte
	isTLS bool

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

func (c *serveConn) upgradeTLS(ctx context.Context, buf []byte) (err error) {
	// TLS in this library follows what I will refer to as the "Lohmann Model":
	// https://lists.debian.org/gopher-project/2018/02/msg00038.html
	//
	// So ASCII 0x16 (SYN) is reserved and is now forbidden as the first character of a
	// selector; if 0x16 is sent, the server presumes it commences a TLS handshake.
	//
	if c.buf[0] != 0x16 || c.isTLS {
		return nil
	}
	if c.srv.TLSConfig == nil {
		// XXX: Non-TLS gopher will typically respond to a request for 0x16 with an
		// directory with a '3' type or a bodgy text error message, which will cause
		// the client to cop a tls.RecordHeaderError, so we should do the same as
		// that's the error we want to catch in the client.
		//
		// We want to have a pluggable error renderer in here too, but we don't want
		// this particular write to be overridden; it's a question for later.
		c.rwc.Write(upgradeTLSErrorResponse)

		return fmt.Errorf("gopher: tls not configured")
	}

	bufConn := &bufferedConn{
		Conn: c.rwc,
		rdr:  io.MultiReader(bytes.NewReader(buf), c.rwc),
	}
	c.isTLS = true
	c.rwc = tls.Server(bufConn, c.srv.TLSConfig)
	return nil
}

func (c *serveConn) readRequest(ctx context.Context) (req *Request, err error) {
retryTLS:
	c.rwc.SetReadDeadline(time.Now().Add(c.srv.readSelectorTimeout()))

	var nl, at, sz int
	for {
		n, err := c.rwc.Read(c.buf[sz:])
		if err != nil && (err != io.EOF || n == 0) {
			return nil, err
		}

		// Only attempt TLS upgrade if this is the first read and haven't already
		// successfully upgraded:
		if sz == 0 && !c.isTLS && n > 0 {
			if err := c.upgradeTLS(ctx, c.buf[:n]); err != nil {
				return nil, err
			}
			if c.isTLS {
				goto retryTLS
			}
		}
		sz += n

		// Scan for the newline from the end of the last read:
		for i := at; i < sz; i++ {
			if c.buf[i] == '\n' {
				nl = i
				goto found
			}
		}
		at = sz
	}

found:
	line, left := c.buf[:nl], c.buf[nl+1:]
	line = dropCR(line)

	var url = URL{Hostname: c.host, Port: c.port}

	fileFlag, err := populateRequestURL(&url, line)
	if err != nil {
		return nil, err
	}

	var body io.ReadCloser = c.rwc
	if len(left) > 0 || fileFlag {
		c.rwc.SetReadDeadline(time.Now().Add(c.srv.readTimeout()))

		multi := io.MultiReader(bytes.NewReader(left), c.rwc)
		body = &readCloser{
			readFn:  multi.Read,
			closeFn: c.rwc.Close,
		}
	}

	rq := NewRequest(url, body)
	rq.RemoteAddr = c.rwc.RemoteAddr().(*net.TCPAddr)

	return rq, nil
}

func dropCR(data []byte) []byte {
	sz := len(data)
	if len(data) > 0 && data[sz-1] == '\r' {
		return data[0 : sz-1]
	}
	return data
}

func populateRequestURL(url *URL, line []byte) (fileFlag bool, err error) {
	var field, s int
	var sz = len(line)

	url.ItemType = Text

	for i := 0; i <= sz; i++ {
		if i == sz || line[i] == '\t' {
			switch field {
			case 0:
				url.Selector = string(line[s:i])
				url.Root = url.Selector == ""
				field, s = field+1, i+1

			case 1:
				url.Search = string(line[s:i])
				field, s = field+1, i+1

			case 2:
				ok := i-s == 1 && (line[s] == '0' || line[s] == '1')
				if !ok {
					// XXX: perhaps invalid file flags should just be ignored?
					return false, ErrRequestFileFlagInvalid
				}
				fileFlag = line[s] == '1'
				field, s = field+1, i+1

			case 3:
				// XXX: Gopher clients could send us any old garbage. Should we ignore
				// and carry on?
				return fileFlag, ErrRequestTrailingData
			}
		}
	}

	return fileFlag, nil
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
