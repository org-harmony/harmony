package web

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

// HRouter is an implementation of Router.
// It uses chi.Router internally and wraps the Router interface as an abstraction around it.
type HRouter struct {
	r chi.Router
}

// Router is an interface for router
// It is composed of http.Handler, VerbRouter and MiddlewareRouter.
// Router itself provides SubRouter, Handle, NotFound, and MethodNotAllowed.
type Router interface {
	http.Handler
	VerbRouter
	MiddlewareRouter

	SubRouter(path string, handler http.Handler)  // SubRouter mounts a sub-router on a path.
	Handle(path string, handler http.HandlerFunc) // Handle registers a handler function.
	NotFound(handler http.HandlerFunc)            // NotFound registers a handler for when a route could not be found.
	MethodNotAllowed(handler http.HandlerFunc)    // MethodNotAllowed registers a handler for when a method is not allowed.
}

// VerbRouter is an interface for router that handles HTTP verbs.
type VerbRouter interface {
	http.Handler
	Get(path string, handler http.HandlerFunc)    // Get registers a handler for GET requests.
	Post(path string, handler http.HandlerFunc)   // Post registers a handler for POST requests.
	Put(path string, handler http.HandlerFunc)    // Put registers a handler for PUT requests.
	Delete(path string, handler http.HandlerFunc) // Delete registers a handler for DELETE requests.
	Patch(path string, handler http.HandlerFunc)  // Patch registers a handler for PATCH requests.
}

// MiddlewareRouter is an interface for router that handles middleware.
type MiddlewareRouter interface {
	http.Handler
	Use(...func(http.Handler) http.Handler)         // Use appends one or more middlewares onto the Router stack.
	With(...func(http.Handler) http.Handler) Router // With adds inline middlewares for an endpoint handler.
}

// NewRouter returns a new Router.
func NewRouter() Router {
	return &HRouter{
		r: chi.NewRouter(),
	}
}

// ServeHTTP implements http.Handler.
func (r *HRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.r.ServeHTTP(w, req)
}

// SubRouter mounts a sub-router on a path.
func (r *HRouter) SubRouter(path string, handler http.Handler) {
	r.r.Mount(path, handler)
}

// Get registers a handler for GET requests.
func (r *HRouter) Get(path string, handler http.HandlerFunc) {
	r.r.Get(path, handler)
}

// Post registers a handler for POST requests.
func (r *HRouter) Post(path string, handler http.HandlerFunc) {
	r.r.Post(path, handler)
}

// Put registers a handler for PUT requests.
func (r *HRouter) Put(path string, handler http.HandlerFunc) {
	r.r.Put(path, handler)
}

// Delete registers a handler for DELETE requests.
func (r *HRouter) Delete(path string, handler http.HandlerFunc) {
	r.r.Delete(path, handler)
}

// Patch registers a handler for PATCH requests.
func (r *HRouter) Patch(path string, handler http.HandlerFunc) {
	r.r.Patch(path, handler)
}

// Handle registers a handler function.
func (r *HRouter) Handle(path string, handler http.HandlerFunc) {
	r.r.Handle(path, handler)
}

// NotFound registers a handler for when a route could not be found.
func (r *HRouter) NotFound(handler http.HandlerFunc) {
	r.r.NotFound(handler)
}

// MethodNotAllowed registers a handler for when a method is not allowed.
func (r *HRouter) MethodNotAllowed(handler http.HandlerFunc) {
	r.r.MethodNotAllowed(handler)
}

// Use appends one or more middlewares onto the Router stack.
func (r *HRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.r.Use(middlewares...)
}

// With adds inline middlewares for an endpoint handler.
func (r *HRouter) With(middlewares ...func(http.Handler) http.Handler) Router {
	return &HRouter{
		r: r.r.With(middlewares...),
	}
}

// URLParam returns the URL parameter by key.
func URLParam(req *http.Request, key string) string {
	return chi.URLParam(req, key)
}
