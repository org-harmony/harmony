package harmony

import (
	"context"
	"net/http"
)

type Server interface {
	Serve(ctx context.Context) error
	RegisterController(c ...Controller)
	RegisterMiddleware(m ...func(http.Handler) http.Handler)
}

type Controller interface{}
