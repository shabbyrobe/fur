package gopher

import (
	"io"
	"time"
)

type Recorder interface {
	BeginRecording(u URL, at time.Time) Recording
}

type Recording interface {
	RequestWriter() io.Writer
	ResponseWriter() io.Writer
	SetStatus(status Status, msg string)
	Done(at time.Time)
}

func recordConn(rec Recording, c conn) conn {
	return &recordedConn{
		conn: c,
		rec:  rec,
		rdr:  io.TeeReader(c, rec.ResponseWriter()),
		wrt:  io.MultiWriter(c, rec.RequestWriter()),
	}
}

type recordedConn struct {
	conn
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
	return rc.conn.Close()
}
