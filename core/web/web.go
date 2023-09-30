// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"context"
	"net/http"
)

const Pkg = "sys.web"

type Server interface {
	Serve(ctx context.Context) error
	RegisterController(c ...Controller)
	RegisterMiddleware(m ...func(http.Handler) http.Handler)
}

type Controller interface{}
