package furball

import (
	"bytes"
	"io"
	"time"

	"github.com/shabbyrobe/fur/internal/gopher"
)

type Ball struct {
	Entries []Entry `json:"entries"`
}

var _ gopher.Recorder = &Ball{}

func (b *Ball) BeginRecording(u gopher.URL, at time.Time) gopher.Recording {
	if b == nil {
		return nil
	}
	return &EntryRecording{
		ball: b,
		entry: Entry{
			URL: u,
			At:  at,
		},
	}
}

type Entry struct {
	URL    gopher.URL    `json:"url"`
	At     time.Time     `json:"at"`
	Taken  Duration      `json:"taken"`
	Status gopher.Status `json:"status,omitempty"`
	Msg    string        `json:"msg,omitempty"`
	In     []byte        `json:"in,omitempty"`
	Out    []byte        `json:"out"`
}

type EntryRecording struct {
	ball  *Ball
	entry Entry
	in    bytes.Buffer
	out   bytes.Buffer
}

func (e *EntryRecording) RequestWriter() io.Writer  { return &e.in }
func (e *EntryRecording) ResponseWriter() io.Writer { return &e.out }

func (e *EntryRecording) SetStatus(status gopher.Status, msg string) {
	e.entry.Status = status
	e.entry.Msg = msg
}

func (e *EntryRecording) Done(at time.Time) {
	e.entry.In = e.in.Bytes()
	e.entry.Out = e.out.Bytes()
	e.entry.Taken = Duration(at.Sub(e.entry.At))
	e.ball.Entries = append(e.ball.Entries, e.entry)
}
