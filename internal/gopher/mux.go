package gopher

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Mux is the 2-hour version of httprouter. Maybe I'll come back and add the 1-week
// version later, but this is reasonably quick and has most of the features.
//
// Slashes ('/') are always trimmed from patterns and search paths.
//
// Named parameters only match a single path segment:
//
// Pattern: /user/:user
//
//  /user/gordon              match
//  /user/you                 match
//  /user/gordon/profile      no match
//  /user/                    no match
//
// Catch-All parameters
//
// Catch-all parameters and have the form *name, and match everything to the end
// of the input. Catch-all parameters must be at the end fo the pattern.
//
// Pattern: /src/*filepath
//
//	/src                      match   <-- (!)
//	/src/                     match
//	/src/somefile.go          match
//	/src/subdir/somefile.go   match
//
// Leading/Trailing Slashes
//
// Mux strips leading and trailing slashes. If the last segment is a catch
// all and the previous segment does not match an existing node, the catch-all
// will receive an empty path.
//
// Patterns: /foo, /foo/*rest
//
//  Selector      Handler
//	/foo          /foo
//	/foo/         /foo/*rest
//	/foo/stuff    /foo/*rest
//
// Patterns: /foo/*rest
//
//  Selector      Handler
//	/foo          /foo/*rest
//	/foo/         /foo/*rest
//	/foo/stuff    /foo/*rest
//
type Mux struct {
	root      muxNode
	maxParams int

	CatchAllRequiresTrailingSlash bool
}

func NewMux() *Mux {
	m := &Mux{}
	return m
}

func (mux *Mux) addParam(params Params, p Param) Params {
	if params == nil {
		params = make(Params, 0, mux.maxParams)
	}
	return append(params, p)
}

func (mux *Mux) findNode(path string) (*muxNode, Params) {
	orig := path
	path = trimSlash(path)
	hasTrailingSlash := len(path) > 0 && orig[len(orig)-1] == '/'

	start := 0

	var params Params
	var lastWild *muxNode
	var lastWildStart, lastWildParam int

	cur := &mux.root
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if start == i {
				start++
				continue // Skip empty segments
			}

			// If see a catch-all, grab hold of it as it's what we should fall back to if
			// the match fails from here.
			if cur.childWild != nil && cur.childWild.kind == muxNodeCatchAll {
				lastWild, lastWildStart, lastWildParam = cur.childWild, start, len(params)
			}

			// Fixed paths take precedence:
			if cur.childPaths != nil {
				next, ok := cur.childPaths[path[start:i]]
				if ok {
					cur = next
					start = i + 1
					continue
				}
			}

			// Now check if we have a param match (skipping over catch-all matches, which
			// we deal with at the end):
			if cur.childWild != nil {
				cur = cur.childWild
				if cur.kind == muxNodeParam {
					params = mux.addParam(params, Param{cur.param, path[start:i]})
				}
				start = i + 1
				continue

			} else {
				cur = nil
				break
			}
		}
	}

	if cur != nil &&
		cur.handler == nil &&
		cur.meta == nil &&
		cur.childWild != nil &&
		cur.childWild.kind == muxNodeCatchAll &&
		(!mux.CatchAllRequiresTrailingSlash || hasTrailingSlash || path == "") {

		// This covers the situation where the found node has no handler, but it has
		// a catch-all as a child.

		cur = cur.childWild
		params = mux.addParam(params, Param{cur.param, ""})

	} else if (cur == nil || cur.kind == muxNodeCatchAll) && lastWild != nil {
		// This covers if we see the catch-all last, or if we see a catch-all at some
		// point during our traversal but the more specific match fails:
		cur = lastWild
		params = mux.addParam(params[:lastWildParam], Param{cur.param, path[lastWildStart:]})
	}

	return cur, params
}

// Handle a pattern with Handler and/or MetaHandler.
//
// Calling Handle() twice with the same pattern will result in a panic.
//
// Patterns are described in Mux's documentation.
//
// If meta is nil but handler implements MetaHandler, it will be used as the MetaHandler.
//
func (mux *Mux) Handle(pattern string, handler Handler, meta MetaHandler) {
	if meta == nil {
		hmeta, ok := handler.(MetaHandler)
		if ok {
			meta = hmeta
		}
	}

	parent := &mux.root

	pattern = trimSlash(pattern)
	if pattern == "" {
		if parent.HasChildren() {
			panic(fmt.Errorf("gopher: root handler already exists"))
		}
		parent.handler, parent.meta = handler, meta
		return
	}

	parts := strings.Split(pattern, "/")
	last := ""
	path, last := parts[:len(parts)-1], parts[len(parts)-1]

	params := 0

	for _, part := range path {
		if len(part) == 0 {
			continue
		}
		switch part[0] {
		case '*':
			panic(errors.New("gopher: mux catch-all must be last"))

		case ':':
			if parent.childWild != nil && parent.childWild.part != part {
				panic(fmt.Errorf("gopher: param %q conflicts with existing param %q for pattern %q", part, parent.childWild.param, pattern))
			}

			if parent.childWild == nil {
				cur := &muxNode{parent: parent, part: part, kind: muxNodeParam, param: part[1:]}
				parent.childWild = cur
			}

			parent = parent.childWild
			params++

		default:
			children := parent.ChildPaths()

			cur, ok := children[part]
			if !ok {
				cur = &muxNode{parent: parent, part: part, kind: muxNodePath}
				children[part] = cur
			}
			parent = cur
		}
	}

	switch last[0] {
	case '*', ':':
		kind := muxNodeParam
		if last[0] == '*' {
			kind = muxNodeCatchAll
		}
		if parent.childWild != nil {
			child := parent.childWild
			if child.HasAnyHandler() {
				panic(fmt.Errorf("gopher: param node %q already has handler for pattern %q", last, pattern))
			}
			child.handler, child.meta = handler, meta

		} else {
			parent.childWild = &muxNode{
				parent: parent, part: last, param: last[1:], kind: kind,
				handler: handler, meta: meta,
			}
		}
		params++

	default:
		children := parent.ChildPaths()
		child := children[last]
		if child != nil {
			if child.HasAnyHandler() {
				panic(fmt.Errorf("gopher: mux path %q already exists", pattern))
			}
			// If the node already exists, but doesn't have a handler, it's safe to set.
			child.kind, child.handler, child.meta = muxNodePath, handler, meta

		} else {
			children[last] = &muxNode{
				parent: parent, part: last, kind: muxNodePath,
				handler: handler, meta: meta}
		}
	}

	mux.updateParamsCap(params)
}

func (mux *Mux) updateParamsCap(params int) {
	const paramsCap = 8
	if params > mux.maxParams {
		mux.maxParams = params
	}
	if mux.maxParams > paramsCap {
		mux.maxParams = paramsCap
	}
}

func (mux *Mux) ServeGopher(ctx context.Context, w ResponseWriter, r *Request) {
	h, params := mux.findNode(r.url.Selector)
	if h == nil || h.handler == nil {
		NotFound(w, r)
		return
	}
	r.Params = params

	h.handler.ServeGopher(ctx, w, r)
}

func (mux *Mux) ServeGopherMeta(ctx context.Context, w MetaWriter, r *Request) {
	h, params := mux.findNode(r.url.Selector)

	if h == nil {
		w.MetaError(StatusNotFound, fmt.Sprintf("Not found: %q", r.url.Selector))
		return
	}
	if h.meta == nil {
		h.meta = metaHandlerDefault
	}
	r.Params = params

	h.meta.ServeGopherMeta(ctx, w, r)
}

var metaHandlerDefault = MetaHandlerFunc(func(ctx context.Context, mw MetaWriter, rq *Request) {
	mw.Info(Text, rq.url.Selector, rq.url.Selector)
})

type muxNode struct {
	part       string
	param      string
	parent     *muxNode
	childWild  *muxNode
	childPaths map[string]*muxNode
	kind       byte

	handler Handler
	meta    MetaHandler
}

func (m *muxNode) HasAnyHandler() bool {
	return m.handler != nil || m.meta != nil
}

func (m *muxNode) HasChildren() bool {
	return m.childWild != nil || (m.childPaths != nil && len(m.childPaths) > 0)
}

func (m *muxNode) HasChildPaths() bool {
	return (m.childPaths != nil && len(m.childPaths) > 0)
}

func (m *muxNode) ChildPaths() map[string]*muxNode {
	if m.childPaths == nil {
		m.childPaths = map[string]*muxNode{}
	}
	return m.childPaths
}

const (
	muxNodePath     byte = 0
	muxNodeParam    byte = 1
	muxNodeCatchAll byte = 2
)

func trimSlash(s string) string {
	if len(s) == 0 {
		return s
	}
	first := 0
	for ; first < len(s); first++ {
		if s[first] != '/' {
			break
		}
	}
	last := len(s) - 1
	for i := last; i >= 0; i-- {
		if s[i] != '/' {
			last = i
			break
		}
	}
	return s[first : last+1]
}
