package web

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony"
	"net/http"
)

// Server contains the configuration for a web server.
// It implements the harmony.Server interface.
type Server struct {
	config *ServerConfig
}

// ServerConfig contains the configuration for a web server.
type ServerConfig struct {
	Router       chi.Router
	Logger       harmony.Logger
	Addr         string
	EventManager *harmony.StdEventManager
}

// ServerConfigFunc is a function that configures a ServerConfig.
// This follows the functional options pattern and is used to configure the web server
type ServerConfigFunc func(*ServerConfig)

// WithRouter configures the router for the web server.
func WithRouter(r chi.Router) ServerConfigFunc {
	return func(cfg *ServerConfig) {
		cfg.Router = r
	}
}

// WithLogger configures the logger for the web server.
func WithLogger(l harmony.Logger) ServerConfigFunc {
	return func(cfg *ServerConfig) {
		cfg.Logger = l
	}
}

// WithAddr configures the address for the web server.
func WithAddr(addr string) ServerConfigFunc {
	return func(cfg *ServerConfig) {
		cfg.Addr = addr
	}
}

// WithEventManger configures the event manager for the web server.
func WithEventManger(em *harmony.StdEventManager) ServerConfigFunc {
	return func(cfg *ServerConfig) {
		cfg.EventManager = em
	}
}

// defaultServerConfig returns a new instance of ServerConfig with default values.
func defaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Router: chi.NewRouter(),
		Logger: harmony.NewStdLogger(),
		Addr:   ":8080",
	}
}

// NewServer creates a new instance of Server.
// The server is configured using the provided ServerConfigFuncs.
// If no ServerConfigFuncs are provided the server is configured with default values from defaultServerConfig.
func NewServer(cfg ...ServerConfigFunc) *Server {
	config := defaultServerConfig()

	for _, f := range cfg {
		f(config)
	}

	return &Server{
		config: config,
	}
}

// Serve starts the web server.
// It blocks until the server is stopped or an error occurs.
// The Serve() can be run in a goroutine to prevent blocking.
func (s *Server) Serve(ctx context.Context) error {
	s.config.Logger.Info(Pkg, "starting web server...")

	return http.ListenAndServe(s.config.Addr, s.config.Router)
}

// RegisterController registers a controller with the web server.
func (s *Server) RegisterController(c ...harmony.Controller) {
	s.config.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello World!")
		if err != nil {
			s.config.Logger.Error(Pkg, "failed to write response", err)
		}
	})
}

// RegisterMiddleware registers middleware with the web server.
func (s *Server) RegisterMiddleware(m ...func(http.Handler) http.Handler) {
	s.config.Router.Use(m...)
}
