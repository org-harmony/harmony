package web

import (
	"context"
	"net/http"
)

const ServerMod = "sys.web.server"

type Server interface {
	Setup(ctx context.Context) error
	Serve(ctx context.Context) error
	RegisterController(c ...Controller)
	RegisterMiddleware(m ...func(http.Handler) http.Handler)
}

type Controller interface{}
