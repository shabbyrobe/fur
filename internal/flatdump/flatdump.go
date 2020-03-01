package flatdump

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/shabbyrobe/furlib/gopher"
)

// FlatDump is a brutally simple way of dumping a Gopher request that I've used for some
// tests here and there and from bash.
//
// Format:
//	FUR-DUMP <iso8601dt> <url>
//	<response>...
//
// To dump from bash (wrecking all trailing newlines in the process):
//	url=...
//	out=...
//  echo -en "FUR-DUMP $( date -Is ) $url\n$out"
//
type FlatDump struct {
	At  time.Time
	URL gopher.URL
	io.Reader
}

func ReadFlatDump(rdr io.Reader) (*FlatDump, error) {
	var buf = make([]byte, 8192)

	n, err := rdr.Read(buf)
	if err != nil && (err != io.EOF || n == 0) {
		return nil, err
	}
	buf = buf[:n]
	nl := bytes.IndexByte(buf, '\n')
	if nl < 0 {
		return nil, fmt.Errorf("furball: no newline found in first read")
	}
	line, rest := buf[:nl], buf[nl+1:]

	if !bytes.HasPrefix(line, furDumpMagic) {
		return nil, fmt.Errorf("furball: no magic found")
	}
	line = bytes.TrimLeft(line[len(furDumpMagic):], " ")

	dateEnd := bytes.IndexByte(line, ' ')
	if dateEnd < 0 {
		return nil, fmt.Errorf("furball: date not found")
	}

	tm, err := time.Parse(time.RFC3339, string(line[:dateEnd]))
	if err != nil {
		return nil, err
	}

	url, err := gopher.ParseURL(string(bytes.TrimRight(line[dateEnd+1:], "\r")))
	if err != nil {
		return nil, err
	}

	return &FlatDump{
		At:     tm,
		URL:    url,
		Reader: io.MultiReader(bytes.NewReader(rest), rdr),
	}, nil
}

func WriteFlatDumpHeader(w io.Writer, url gopher.URL, at time.Time) (n int, err error) {
	return fmt.Fprintf(w, "FUR-DUMP %s %s\n", at.Format(time.RFC3339), url)
}

var (
	furDumpMagic = []byte("FUR-DUMP")
)
