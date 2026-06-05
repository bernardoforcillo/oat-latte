package oat

import "strings"

// RouteParams holds named path segments extracted during route matching.
// Pattern "/users/:id" matched against "/users/42" yields {"id": "42"}.
type RouteParams map[string]string

// Navigator provides navigation operations to ScreenBuilder functions.
type Navigator interface {
	Navigate(path string)
	GoBack()
	CurrentPath() string
	CanGoBack() bool
}

// ScreenBuilder is a factory that returns the body Component for a route.
// It receives extracted path params and a Navigator for further navigation.
type ScreenBuilder func(params RouteParams, nav Navigator) Component

// Router maps URL-style path patterns to ScreenBuilders.
// Patterns are matched in registration order; first match wins.
type Router struct {
	routes []routeEntry
}

type routeEntry struct {
	pattern  string
	segments []string // pattern split on "/"
	builder  ScreenBuilder
}

// NewRouter creates an empty Router.
func NewRouter() *Router { return &Router{} }

// Handle registers builder for the given URL pattern.
// Pattern segments starting with ":" are named parameters.
// Example patterns: "/", "/users", "/users/:id", "/users/:id/posts/:postId"
func (r *Router) Handle(pattern string, builder ScreenBuilder) *Router {
	segs := splitPath(pattern)
	r.routes = append(r.routes, routeEntry{pattern: pattern, segments: segs, builder: builder})
	return r
}

// Match returns the builder and extracted params for path, or (nil, nil, false).
func (r *Router) Match(path string) (ScreenBuilder, RouteParams, bool) {
	pathSegs := splitPath(path)
	for _, route := range r.routes {
		if len(route.segments) != len(pathSegs) {
			continue
		}
		params := RouteParams{}
		matched := true
		for i, seg := range route.segments {
			if strings.HasPrefix(seg, ":") {
				params[seg[1:]] = pathSegs[i]
			} else if seg != pathSegs[i] {
				matched = false
				break
			}
		}
		if matched {
			return route.builder, params, true
		}
	}
	return nil, nil, false
}

// splitPath splits a URL path on "/" removing empty segments.
func splitPath(p string) []string {
	parts := strings.Split(p, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
