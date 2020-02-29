package gopher

import (
	"io"
	"net"
	"time"
)

type Recorder interface {
	BeginRecording(rq *Request, at time.Time) Recording
}

type Recording interface {
	RequestWriter() io.Writer
	ResponseWriter() io.Writer
	SetStatus(status Status, msg string)
	Done(at time.Time)
}

func recordConn(rec Recording, c net.Conn) net.Conn {
	return &recordedConn{
		Conn: c,
		rec:  rec,
		rdr:  io.TeeReader(c, rec.ResponseWriter()),
		wrt:  io.MultiWriter(c, rec.RequestWriter()),
	}
}

type recordedConn struct {
	net.Conn
	rec Recording
	rdr io.Reader
	wrt io.Writer
}

func (rc *recordedConn) Read(b []byte) (n int, err error) {
	return rc.rdr.Read(b)
}

func (rc *recordedConn) Write(b []byte) (n int, err error) {
	return rc.wrt.Write(b)
}

func (rc *recordedConn) Close() error {
	rc.rec.Done(time.Now())
	return rc.Conn.Close()
}
