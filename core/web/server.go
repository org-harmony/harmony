package web

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"net/http"
)

// StdServer contains the configuration for a web server.
// It implements the Server interface.
type StdServer struct {
	config *ServerConfigs
}

// ServerConfigs contains the configuration for a web server.
type ServerConfigs struct {
	Router       chi.Router
	Logger       trace.Logger
	Addr         string
	EventManager *event.StdEventManager
}

// ServerConfig is a function that configures a ServerConfigs.
// This follows the functional options pattern and is used to configure the web server
type ServerConfig func(*ServerConfigs)

// WithRouter configures the router for the web server.
func WithRouter(r chi.Router) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Router = r
	}
}

// WithLogger configures the logger for the web server.
func WithLogger(l trace.Logger) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Logger = l
	}
}

// WithAddr configures the address for the web server.
func WithAddr(addr string) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Addr = addr
	}
}

// WithEventManger configures the event manager for the web server.
func WithEventManger(em *event.StdEventManager) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.EventManager = em
	}
}

// defaultServerConfigs returns a new instance of ServerConfigs with default values.
func defaultServerConfigs() *ServerConfigs {
	return &ServerConfigs{
		Router: chi.NewRouter(),
		Logger: trace.NewLogger(),
		Addr:   ":8080",
	}
}

// NewServer creates a new instance of StdServer.
// The server is configured using the provided ServerConfig functions.
// If no ServerConfig functions are provided the server is configured with default values from defaultServerConfigs.
func NewServer(cfg ...ServerConfig) *StdServer {
	config := defaultServerConfigs()

	for _, f := range cfg {
		f(config)
	}

	return &StdServer{
		config: config,
	}
}

// Serve starts the web server.
// It blocks until the server is stopped or an error occurs.
// Serve can be run in a goroutine to prevent blocking.
func (s *StdServer) Serve(ctx context.Context) error {
	s.config.Logger.Info(Pkg, "starting web server...")

	return http.ListenAndServe(s.config.Addr, s.config.Router)
}

// RegisterController registers a controller with the web server.
func (s *StdServer) RegisterController(c ...Controller) {
	s.config.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello World!")
		if err != nil {
			s.config.Logger.Error(Pkg, "failed to write response", err)
		}
	})
}

// RegisterMiddleware registers middleware with the web server.
func (s *StdServer) RegisterMiddleware(m ...func(http.Handler) http.Handler) {
	s.config.Router.Use(m...)
}
