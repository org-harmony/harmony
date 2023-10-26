package web

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

// HRouter uses chi.Router internally and wraps the Router interface as an abstraction around it.
type HRouter struct {
	r chi.Router
}

// Router is composed of http.Handler, VerbRouter and MiddlewareRouter.
// It is meant as an abstraction around chi.Router. But it can be implemented by any router.
type Router interface {
	http.Handler
	VerbRouter
	MiddlewareRouter

	SubRouter(path string, handler http.Handler)
	Handle(path string, handler http.HandlerFunc)
	NotFound(handler http.HandlerFunc)
	MethodNotAllowed(handler http.HandlerFunc)
}

// VerbRouter is an interface for router that handles HTTP verbs.
type VerbRouter interface {
	http.Handler
	Get(path string, handler http.HandlerFunc)
	Post(path string, handler http.HandlerFunc)
	Put(path string, handler http.HandlerFunc)
	Delete(path string, handler http.HandlerFunc)
	Patch(path string, handler http.HandlerFunc)
}

// MiddlewareRouter is an interface for router that handles middleware.
type MiddlewareRouter interface {
	http.Handler
	Use(...func(http.Handler) http.Handler)         // Use appends one or more middlewares onto the Router stack.
	With(...func(http.Handler) http.Handler) Router // With adds inline middlewares for an endpoint handler.
}

func NewRouter() Router {
	return &HRouter{
		r: chi.NewRouter(),
	}
}

func (r *HRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

func (r *HRouter) SubRouter(path string, handler http.Handler) {
	r.r.Mount(path, handler)
}

func (r *HRouter) Get(path string, handler http.HandlerFunc) {
	r.r.Get(path, handler)
}

func (r *HRouter) Post(path string, handler http.HandlerFunc) {
	r.r.Post(path, handler)
}

func (r *HRouter) Put(path string, handler http.HandlerFunc) {
	r.r.Put(path, handler)
}

func (r *HRouter) Delete(path string, handler http.HandlerFunc) {
	r.r.Delete(path, handler)
}

func (r *HRouter) Patch(path string, handler http.HandlerFunc) {
	r.r.Patch(path, handler)
}

func (r *HRouter) Handle(path string, handler http.HandlerFunc) {
	r.r.Handle(path, handler)
}

func (r *HRouter) NotFound(handler http.HandlerFunc) {
	r.r.NotFound(handler)
}

func (r *HRouter) MethodNotAllowed(handler http.HandlerFunc) {
	r.r.MethodNotAllowed(handler)
}

func (r *HRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.r.Use(middlewares...)
}

func (r *HRouter) With(middlewares ...func(http.Handler) http.Handler) Router {
	return &HRouter{
		r: r.r.With(middlewares...),
	}
}

func URLParam(req *http.Request, key string) string {
	return chi.URLParam(req, key)
}
