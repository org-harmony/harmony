package web

import (
	"github.com/go-chi/chi/v5/middleware"
)

var (
	// CleanPath middleware will clean out double slashes from the request URL. It is a wrapper for middleware.CleanPath.
	CleanPath = middleware.CleanPath
	// Heartbeat creates a heartbeat endpoint. It is a wrapper for middleware.Heartbeat.
	Heartbeat = middleware.Heartbeat
	// Recoverer middleware recovers from panics and writes a 500 status if there was one. It is a wrapper for middleware.Recoverer.
	Recoverer = middleware.Recoverer
)
