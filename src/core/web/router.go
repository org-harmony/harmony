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

// NewRouter constructs a new Router using chi.Router internally.
func NewRouter() Router {
	return &HRouter{
		r: chi.NewRouter(),
	}
}

// ServeHTTP implements http.Handler by calling the underlying chi.Router's ServeHTTP.
func (r *HRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

// SubRouter implements Router.SubRouter by calling the underlying chi.Router's Mount.
// It mounts a sub-router on the given path.
func (r *HRouter) SubRouter(path string, handler http.Handler) {
	r.r.Mount(path, handler)
}

// Get implements VerbRouter.Get by calling the underlying chi.Router's Get and therefore handles GET requests.
func (r *HRouter) Get(path string, handler http.HandlerFunc) {
	r.r.Get(path, handler)
}

// Post implements VerbRouter.Post by calling the underlying chi.Router's Post and therefore handles POST requests.
func (r *HRouter) Post(path string, handler http.HandlerFunc) {
	r.r.Post(path, handler)
}

// Put implements VerbRouter.Put by calling the underlying chi.Router's Put and therefore handles PUT requests.
func (r *HRouter) Put(path string, handler http.HandlerFunc) {
	r.r.Put(path, handler)
}

// Delete implements VerbRouter.Delete by calling the underlying chi.Router's Delete and therefore handles DELETE requests.
func (r *HRouter) Delete(path string, handler http.HandlerFunc) {
	r.r.Delete(path, handler)
}

// Patch implements VerbRouter.Patch by calling the underlying chi.Router's Patch and therefore handles PATCH requests.
func (r *HRouter) Patch(path string, handler http.HandlerFunc) {
	r.r.Patch(path, handler)
}

// Handle implements Router.Handle by calling the underlying chi.Router's Handle.
func (r *HRouter) Handle(path string, handler http.HandlerFunc) {
	r.r.Handle(path, handler)
}

// NotFound implements Router.NotFound by calling the underlying chi.Router's NotFound.
func (r *HRouter) NotFound(handler http.HandlerFunc) {
	r.r.NotFound(handler)
}

// MethodNotAllowed implements Router.MethodNotAllowed by calling the underlying chi.Router's MethodNotAllowed.
func (r *HRouter) MethodNotAllowed(handler http.HandlerFunc) {
	r.r.MethodNotAllowed(handler)
}

// Use implements MiddlewareRouter.Use by calling the underlying chi.Router's Use.
// It appends one or more middlewares onto the Router stack.
func (r *HRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.r.Use(middlewares...)
}

// With implements MiddlewareRouter.With by calling the underlying chi.Router's With.
// It adds inline middlewares for an endpoint handler and returns a new Router.
func (r *HRouter) With(middlewares ...func(http.Handler) http.Handler) Router {
	return &HRouter{
		r: r.r.With(middlewares...),
	}
}

// URLParam returns the URL parameter from the request. E.g. /users/{id} -> URLParam(req, "id").
func URLParam(req *http.Request, key string) string {
	return chi.URLParam(req, key)
}
