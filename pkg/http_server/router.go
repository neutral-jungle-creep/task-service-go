package http_server

import (
	"net/http"
	"strings"
)

type Router struct {
	routes map[string]map[string]http.Handler
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]http.Handler),
	}
}

func (r *Router) Register(method, path string, handler http.Handler) {
	method = strings.ToUpper(method)

	if r.routes[path] == nil {
		r.routes[path] = make(map[string]http.Handler)
	}
	r.routes[path][method] = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	handlersByMethod, ok := r.routes[path]
	if ok {
		handler, ok := handlersByMethod[method]
		if ok {
			handler.ServeHTTP(w, req)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
