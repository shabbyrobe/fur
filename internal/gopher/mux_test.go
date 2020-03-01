package gopher

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestMuxSimpleRootHandler(t *testing.T) {
	ptn := ""
	testAllHandled(t, handled, ptn, "")
	testAllHandled(t, missed, ptn, "yep")
}

func TestMuxSimplePathHandler(t *testing.T) {
	testAllHandled(t, handled, "foo/bar", "foo/bar")
	testAllHandled(t, missed, "foo", "foo/bar")
	testAllHandled(t, missed, "foo/bar", "foo")
	testAllHandled(t, missed, "foo/bar", "")
}

func TestMuxHandle(t *testing.T) {
	m := NewMux()
	var dh dummyHandler
	m.Handle("/foo/:bar/yep/:oi", nilHandler, nil)
	m.Handle("/foo/:bar", &dh, nil)
	result, _ := m.findNode("/foo/yep")
	if result.handler != &dh {
		t.Fatal()
	}
}

func TestMuxHandleAddCatchAllOverParamFails(t *testing.T) {
	assertPanic(t, func() {
		m := NewMux()
		m.Handle("/foo/:bar", nilHandler, nil)
		m.Handle("/foo/*bar", nilHandler, nil)
	})
}

func TestMuxHandleAddParamOverParamFails(t *testing.T) {
	assertPanic(t, func() {
		m := NewMux()
		m.Handle("/foo/:bar", nilHandler, nil)
		m.Handle("/foo/:bar", nilHandler, nil)
	})
}

func TestMuxHandleAddPathUnderParam(t *testing.T) {
	var d1, d2 dummyHandler
	m := NewMux()
	m.Handle("/foo/:bar", &d1, nil)
	m.Handle("/foo/:bar/baz", &d2, nil)
	assertNode(t, m, "/foo/1", &d1)
	assertNode(t, m, "/foo/1/baz", &d2)
}

func TestMuxHandleAddPathOverParam(t *testing.T) {
	var d1, d2 dummyHandler

	{
		m := NewMux()
		m.Handle("/foo/bar", &d1, nil)
		m.Handle("/foo/:bar", &d2, nil)
		assertNode(t, m, "/foo/bar", &d1)
		assertNode(t, m, "/foo/wat", &d2)
	}

	{
		m := NewMux()
		m.Handle("/foo/:bar", &d1, nil)
		m.Handle("/foo/bar", &d2, nil)
		assertNode(t, m, "/foo/wat", &d1)
		assertNode(t, m, "/foo/bar", &d2)
	}
}

func TestMuxParamHandler(t *testing.T) {
	testAllHandled(t, handledParams{{"foo", "yep"}}, ":foo", "yep")
	testAllHandled(t, handledParams{{"yep", "bar"}}, "foo/:yep", "foo/bar")

	testAllHandled(t,
		handledParams{{"yep", "bar"}, {"roc", "baz"}},
		"foo/:yep/:roc", "foo/bar/baz")
	testAllHandled(t,
		handledParams{{"a", "foo"}, {"b", "bar"}, {"c", "baz"}},
		":a/:b/:c", "foo/bar/baz")
	testAllHandled(t,
		handledParams{{"a", "foo"}, {"a", "bar"}, {"a", "baz"}},
		":a/:a/:a", "foo/bar/baz")

	testAllHandled(t, missed, ":foo/:bar/:baz", "a//c")

	testAllHandled(t,
		handledParams{{"a", "foo"}, {"b", "bar"}, {"c", "baz"}},
		":a/:b/:c", "foo//bar//baz")

	testAllHandled(t,
		handledParams{{"p1", "val1"}},
		"foo/:p1/baz", "foo/val1/baz",
		otherRoute("foo/:p1/qux"))

	testAllHandled(t,
		handledParams{{"p1", "val1"}, {"p2", "val2"}},
		"foo/:p1/baz/:p2", "foo/val1/baz/val2",
		otherRoute("foo/:p1/qux/:p2"))

	testAllHandled(t, missed, ":p1", "")
	testAllHandled(t, missed, ":p1/baz", "val1")
	testAllHandled(t, missed, ":p1/baz", "val1/")
	testAllHandled(t, missed, ":p1/baz/", "val1/")
	testAllHandled(t, missed, ":p1/baz/", "val1/wat/")
	testAllHandled(t, missed, "foo/:p1", "foo/")
}

func TestMuxCatchAll(t *testing.T) {
	testAllHandled(t, handledParams{{"p1", ""}}, "*p1", "")

	testAllHandled(t, handledParams{{"p1", "val1"}}, "*p1", "val1")
	testAllHandled(t, handledParams{{"p1", "val1/etc"}}, "*p1", "val1/etc")
	testAllHandled(t, handledParams{{"p1", "val1/etc/etc"}}, "*p1", "val1/etc/etc")

	testAllHandled(t, handledParams{{"p1", ""}},
		"foo/*p1", "foo")
	testAllHandled(t, handledParams{{"p1", ""}},
		"foo/*p1", "foo/")
	testAllHandled(t, handledParams{{"p1", "val1"}},
		"foo/*p1", "foo/val1")
	testAllHandled(t, handledParams{{"p1", "val1/etc"}},
		"foo/*p1", "foo/val1/etc")
	testAllHandled(t, handledParams{{"p1", "val1/etc/etc"}},
		"foo/*p1", "foo/val1/etc/etc")

	testAllHandled(t, missed,
		"*p1", "foo/bar", otherRoute("foo/bar"))
	testAllHandled(t, handledParams{{"p1", "foo/baz"}},
		"*p1", "foo/baz", otherRoute("foo/bar"))

	testAllHandled(t, handledParams{{"p1", "qux"}},
		"foo/bar/baz/*p1", "foo/bar/baz/qux",
		otherRoute("foo/bar/*p1"),
		otherRoute("foo/*p1"),
		otherRoute("*p1"),
	)

	testAllHandled(t, handledParams{{"p1", "fleeb/qux"}},
		"foo/bar/*p1", "foo/bar/fleeb/qux",
		otherRoute("foo/bar/baz/*p1"),
		otherRoute("foo/*p1"),
		otherRoute("*p1"),
	)
}

func TestTrimSlash(t *testing.T) {
	for idx, tc := range []struct {
		in  string
		out string
	}{
		{"", ""},
		{"a", "a"},
		{"aa", "aa"},
		{"a/a", "a/a"},
		{"a//a", "a//a"},

		{"/", ""}, {"//", ""}, {"///", ""},
		{"a/", "a"}, {"a//", "a"}, {"a///", "a"},
		{"/a", "a"}, {"//a", "a"}, {"///a", "a"},
		{"/a/", "a"}, {"//a//", "a"}, {"///a///", "a"},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			result := trimSlash(tc.in)
			if result != tc.out {
				t.Fatalf("%q != %q", result, tc.out)
			}
		})
	}
}

type handleResult interface {
	check(t *testing.T, ptn, sel string, handled bool, params Params)
}

type handledStatus bool

func (v handledStatus) check(t *testing.T, ptn, sel string, handled bool, params Params) {
	t.Helper()
	if handled != bool(v) {
		t.Fatalf("selector %q expected handled:%v, found %v", sel, v, handled)
	}
}

type handledParams Params

func (v handledParams) check(t *testing.T, ptn, sel string, handled bool, params Params) {
	t.Helper()
	if !handled {
		t.Fatalf("selector %q expected handled, found %v", sel, handled)
	}
	if !reflect.DeepEqual(params, Params(v)) {
		t.Fatalf("selector %q expected params %v, found %v", sel, v, params)
	}
}

const (
	handled handledStatus = true
	missed  handledStatus = false
)

func testAllHandled(t *testing.T, expected handleResult, ptn string, sel string, otherDefs ...route) {
	testHandled(t, expected, ptn, sel, otherDefs...)
	testMetaHandled(t, expected, ptn, sel, otherDefs...)
}

func testHandled(t *testing.T, expected handleResult, ptn string, sel string, otherDefs ...route) {
	_, file, line, _ := runtime.Caller(2)

	t.Run(fmt.Sprintf("%s:%d", filepath.Base(file), line), func(t *testing.T) {
		var dh dummyHandler

		rq := NewRequest(URL{Selector: sel}, nil)
		mux := applyDefs(NewMux(), otherDefs...)
		mux.Handle(ptn, &dh, nil)
		mux.ServeGopher(context.Background(), &bytes.Buffer{}, rq)

		expected.check(t, ptn, sel, dh.called, dh.params)
	})
}

func testMetaHandled(t *testing.T, expected handleResult, ptn string, sel string, otherDefs ...route) {
	_, file, line, _ := runtime.Caller(2)

	t.Run(fmt.Sprintf("%s:%d", filepath.Base(file), line), func(t *testing.T) {
		var dh dummyHandler

		rq := NewRequest(URL{Selector: sel}.AsMetaItem(), nil)
		mux := applyDefs(NewMux(), otherDefs...)
		mux.Handle(ptn, nil, &dh)
		mw := newMetaWriter(&bytes.Buffer{}, rq)
		mux.ServeGopherMeta(context.Background(), mw, rq)

		expected.check(t, ptn, sel, dh.called, dh.params)
	})
}

func assertNode(t *testing.T, m *Mux, sel string, h Handler) {
	t.Helper()
	n, _ := m.findNode(sel)
	if n == nil || n.handler != h {
		t.Fatal()
	}
}

func assertPanic(t *testing.T, fn func()) {
	defer func() {
		x := recover()
		if x == nil {
			t.Fatal()
		}
	}()
	fn()
}

type route struct {
	path    string
	handler Handler
	meta    MetaHandler
}

func otherRoute(path string) route {
	return route{path, HandlerFunc(nilHandler), MetaHandlerFunc(nilMetaHandler)}
}

func applyDefs(mux *Mux, defs ...route) *Mux {
	for _, rd := range defs {
		mux.Handle(rd.path, rd.handler, rd.meta)
	}
	return mux
}

type dummyHandler struct {
	called bool
	params Params
}

func (d *dummyHandler) assertCalled(b *testing.B) {
	b.Helper()
	if !d.called {
		b.Fatal()
	}
}

func (d *dummyHandler) reset() {
	d.called = false
	d.params = nil
}

func (d *dummyHandler) ServeGopher(ctx context.Context, w ResponseWriter, r *Request) {
	d.called = true
	d.params = r.Params
}

func (d *dummyHandler) ServeGopherMeta(ctx context.Context, w MetaWriter, r *Request) {
	d.called = true
	d.params = r.Params
}

var nilHandler = HandlerFunc(func(context.Context, ResponseWriter, *Request) {})

var nilMetaHandler = MetaHandlerFunc(func(context.Context, MetaWriter, *Request) {})
