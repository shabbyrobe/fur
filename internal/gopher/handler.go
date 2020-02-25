package gopher

import "context"

type Handler interface {
	ServeGopher(ctx context.Context, w ResponseWriter, r *Request)
}

type HandlerFunc func(context.Context, ResponseWriter, *Request)

func (fn HandlerFunc) ServeGopher(ctx context.Context, w ResponseWriter, r *Request) {
	fn(ctx, w, r)
}

type MetaHandler interface {
	ServeGopherMeta(ctx context.Context, w MetaWriter, r *Request)
}

type MetaHandlerFunc func(context.Context, MetaWriter, *Request)

func (fn MetaHandlerFunc) ServeGopherMeta(ctx context.Context, w MetaWriter, r *Request) {
	fn(ctx, w, r)
}
