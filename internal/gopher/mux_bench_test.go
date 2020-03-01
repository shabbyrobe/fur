package gopher

import (
	"context"
	"io/ioutil"
	"testing"
)

func BenchmarkMux_Param(b *testing.B) {
	mux := NewMux()
	mux.Handle("/:a", nilHandler, nil)

	benchRequest(b, mux, "/test")
}

func BenchmarkMux_Param5(b *testing.B) {
	mux := NewMux()
	mux.Handle("/:a/:b/:c/:d/:e", nilHandler, nil)

	benchRequest(b, mux, "/test/test/test/test/test")
}

func BenchmarkMux_GPlusStatic(b *testing.B) {
	var h dummyHandler
	defer h.assertCalled(b)

	mux := NewMux()
	for _, path := range gplusAPI {
		mux.Handle(path, &h, nil)
	}
	benchRequest(b, mux, "/people")
}

func BenchmarkMux_GPlusParam(b *testing.B) {
	var h dummyHandler
	defer h.assertCalled(b)

	mux := NewMux()
	for _, path := range gplusAPI {
		mux.Handle(path, &h, nil)
	}
	benchRequest(b, mux, "/people/118051310819094153327")
}

func BenchmarkMux_GPlus2Params(b *testing.B) {
	var h dummyHandler
	defer h.assertCalled(b)

	mux := NewMux()
	for _, path := range gplusAPI {
		mux.Handle(path, &h, nil)
	}
	benchRequest(b, mux, "/people/118051310819094153327/activities/123456789")
}

func BenchmarkMux_GPlusAll(b *testing.B) {
	mux := NewMux()
	for _, path := range gplusAPI {
		mux.Handle(path, nilHandler, nil)
	}
	benchRoutes(b, mux, gplusAPI)
}

func benchRequest(b *testing.B, mux *Mux, sel string) {
	var w ResponseWriter = ioutil.Discard

	ctx := context.Background()
	rq := NewRequest(URL{Selector: sel}, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mux.ServeGopher(ctx, w, rq)
	}
}

func benchRoutes(b *testing.B, mux *Mux, sels []string) {
	var w ResponseWriter = ioutil.Discard

	ctx := context.Background()
	rq := NewRequest(URL{Selector: ""}, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, sel := range sels {
			rq.url.Selector = sel
			mux.ServeGopher(ctx, w, rq)
		}
	}
}

// Google+
// https://developers.google.com/+/api/latest/
// (in reality this is just a subset of a much larger API)
var gplusAPI = []string{
	"/people/:userId",
	"/people",
	"/activities/:activityId/people/:collection",
	"/people/:userId/people/:collection",
	"/people/:userId/openIdConnect",

	// Activities
	"/people/:userId/activities/:collection",
	"/activities/:activityId",
	"/activities",

	// Comments
	"/activities/:activityId/comments",
	"/comments/:commentId",

	// Moments
	"/people/:userId/moments/:collection",
	"/moments/:id",
}
