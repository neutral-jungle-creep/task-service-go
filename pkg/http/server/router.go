package server

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const paramsContextKey contextKey = "params"

type Router struct {
	routes map[string][]route
}

type route struct {
	pattern string
	handler http.HandlerFunc
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string][]route),
	}
}

func (r *Router) Register(method, pattern string, handler http.HandlerFunc) {
	method = strings.ToUpper(method)

	if len(r.routes[method]) == 0 {
		r.routes[method] = make([]route, 0)
	}
	r.routes[method] = append(r.routes[method], route{pattern: pattern, handler: handler})
}

func matchPattern(pattern, path string) (bool, map[string]string) {
	if !strings.Contains(pattern, "{") && !strings.Contains(pattern, "}") {
		if pattern == path {
			return true, nil
		}
		return false, nil
	}

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false, nil
	}

	var params map[string]string
	for i, p := range patternParts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			if params == nil {
				params = make(map[string]string)
			}
			params[strings.Trim(p, "{}")] = pathParts[i]
			continue
		}
		if p != pathParts[i] {
			return false, nil
		}
	}
	return true, params
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	if routesForMethod, ok := r.routes[method]; ok {
		for _, ro := range routesForMethod {
			if matched, params := matchPattern(ro.pattern, path); matched {
				req = addParamsToContext(req, params)
				ro.handler(w, req)
				return
			}
		}
	}

	// Path may exist under a different method — then it is 405.
	// Otherwise 404.
	for otherMethod, routes := range r.routes {
		if otherMethod == method {
			continue
		}
		for _, ro := range routes {
			if matched, _ := matchPattern(ro.pattern, path); matched {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func addParamsToContext(req *http.Request, params map[string]string) *http.Request {
	if len(params) == 0 {
		return req
	}
	ctx := context.WithValue(req.Context(), paramsContextKey, params)
	return req.WithContext(ctx)
}

func RequestParams(r *http.Request) map[string]string {
	params, ok := r.Context().Value(paramsContextKey).(map[string]string)
	if ok {
		return params
	}
	return nil
}
