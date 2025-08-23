package server

import (
	"context"
	"net/http"
	"strings"
)

const paramsContextKey = "params"

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
	if !strings.Contains(pattern, "{") && !strings.Contains(pattern, "}") && pattern == path {
		return true, nil // это запрос post или get без параметров
	}

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false, nil
	}

	params := make(map[string]string)

	for i := 0; i < len(patternParts); i++ {
		if strings.HasPrefix(patternParts[i], "{") && strings.HasSuffix(patternParts[i], "}") {
			key := strings.Trim(patternParts[i], "{}")
			params[key] = pathParts[i]
		}
	}
	return true, params
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	routesForMethod, ok := r.routes[method]
	if ok {
		for _, ro := range routesForMethod {
			matched, params := matchPattern(ro.pattern, path)
			if matched {
				req = addParamsToContext(req, params)
				ro.handler(w, req)
				return
			}
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
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
