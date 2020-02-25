package gopher

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMetaWriterOneInfoOnly(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaItem(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep", "sel")
	MustFlush(mw)

	expected := "+INFO: 0yep\tsel\tlocalhost\t12345+\r\n"
	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterMultipleInfoOnly(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaDir(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep1", "sel1")
	mw.Info(Dir, "yep2", "sel2")
	mw.Info(Binary, "yep3", "sel3")
	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep1\tsel1\tlocalhost\t12345+" + "\r\n\r\n" +
		"+INFO: 1yep2\tsel2\tlocalhost\t12345+" + "\r\n\r\n" +
		"+INFO: 9yep3\tsel3\tlocalhost\t12345+" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterOneInfoWithOneRecord(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaItem(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep1", "sel1")

	vw := mw.BeginRecord("QUACK")
	if vw == nil {
		t.Fatal()
	}
	vw.WriteLine("hello")
	vw.WriteLine("world")
	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep1\tsel1\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK:\r\nhello\r\nworld" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterOneInfoWithMultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaItem(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep1", "sel1")

	if !mw.WriteRecord("QUACK1", "yep1") {
		t.Fatal()
	}
	if !mw.WriteRecord("QUACK2", "yep2") {
		t.Fatal()
	}
	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep1\tsel1\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK1:\r\nyep1" + "\r\n\r\n" +
		"+QUACK2:\r\nyep2" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterMultipleInfoWithMultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaDir(), nil)
	mw := newMetaWriter(&buf, rq)

	mw.Info(Text, "yep1", "sel1")
	if !mw.WriteRecord("QUACK1", "yep1") {
		t.Fatal()
	}
	if !mw.WriteRecord("QUACK2", "yep2") {
		t.Fatal()
	}

	mw.Info(Dir, "yep2", "sel2")
	if !mw.WriteRecord("QUACK3", "yep3") {
		t.Fatal()
	}
	if !mw.WriteRecord("QUACK4", "yep4") {
		t.Fatal()
	}

	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep1\tsel1\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK1:\r\nyep1" + "\r\n\r\n" +
		"+QUACK2:\r\nyep2" + "\r\n\r\n" +
		"" +
		"+INFO: 1yep2\tsel2\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK3:\r\nyep3" + "\r\n\r\n" +
		"+QUACK4:\r\nyep4" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterValueNormalisesCRLF(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaItem(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep", "sel")

	vw := mw.BeginRecord("QUACK")
	if vw == nil {
		t.Fatal()
	}
	vw.WriteString("line1\n")
	vw.WriteString("line2\n")

	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep\tsel\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK:" + "\r\n" +
		"line1" + "\r\n" +
		"line2" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterValueCRLFOverWriteBoundary(t *testing.T) {
	var buf bytes.Buffer
	var rq = NewRequest(mustParseURL("gopher://localhost:12345").AsMetaItem(), nil)
	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep", "sel")

	vw := mw.BeginRecord("QUACK")
	if vw == nil {
		t.Fatal()
	}
	vw.WriteString("line1")
	vw.WriteString("\r")
	vw.WriteString("\n")
	vw.WriteString("line2")

	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep\tsel\tlocalhost\t12345+" + "\r\n\r\n" +
		"+QUACK:" + "\r\n" +
		"line1" + "\r\n" +
		"line2" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}

func TestMetaWriterValueCRLFExcludesRecords(t *testing.T) {
	var url = mustParseURL("gopher://localhost:12345").AsMetaItem("FOO", "BAR")
	var rq = NewRequest(url, nil)
	var buf bytes.Buffer

	mw := newMetaWriter(&buf, rq)
	mw.Info(Text, "yep", "sel")

	if !mw.WriteRecord("FOO", "yep") {
		t.Fatal("FOO")
	}
	if !mw.WriteRecord("BAR", "yep") {
		t.Fatal("BAR")
	}
	if mw.WriteRecord("BAZ", "nup") {
		t.Fatal()
	}
	if mw.WriteRecord("QUX", "nup") {
		t.Fatal()
	}

	MustFlush(mw)

	expected := "" +
		"+INFO: 0yep\tsel\tlocalhost\t12345+" + "\r\n\r\n" + // INFO should not be excluded
		"+FOO:" + "\r\n" +
		"yep" + "\r\n\r\n" +
		"+BAR:" + "\r\n" +
		"yep" + "\r\n"

	if buf.String() != expected {
		t.Fatal(fmt.Sprintf("%q", buf.String()))
	}
}
