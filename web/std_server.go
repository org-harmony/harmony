package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/core"
	"github.com/org-harmony/harmony/trace"
	"net/http"
)

const StdServerMod = "sys.web.std-server"

type StdServer struct {
	config *StdServerConfig
}

type StdServerConfig struct {
	Router       chi.Router
	Logger       trace.Logger
	Addr         string
	EventManager *core.StdEventManager
}

type StdServerConfigFunc func(*StdServerConfig)

func WithRouter(r chi.Router) StdServerConfigFunc {
	return func(cfg *StdServerConfig) {
		cfg.Router = r
	}
}

func WithLogger(l trace.Logger) StdServerConfigFunc {
	return func(cfg *StdServerConfig) {
		cfg.Logger = l
	}
}

func WithAddr(addr string) StdServerConfigFunc {
	return func(cfg *StdServerConfig) {
		cfg.Addr = addr
	}
}

func WithEventManger(em *core.StdEventManager) StdServerConfigFunc {
	return func(cfg *StdServerConfig) {
		cfg.EventManager = em
	}
}

func defaultStdServerConfig() *StdServerConfig {
	return &StdServerConfig{
		Router: chi.NewRouter(),
		Logger: trace.NewStdLogger(),
		Addr:   ":8080",
	}
}

// NewStdServer creates a new instance of StdServer.
// The server is configured using the provided ServerConfigFuncs.
// If no ServerConfigFuncs are provided the server is configured with default values from defaultStdServerConfig.
//
// The StdEventManager is required if it is not set the function will panic as there is not sense in running the server without an StdEventManager.
func NewStdServer(cfg ...StdServerConfigFunc) *StdServer {
	config := defaultStdServerConfig()

	for _, f := range cfg {
		f(config)
	}

	if config.EventManager == nil {
		panic("event manager is nil but should always be set")
	}

	return &StdServer{
		config: config,
	}
}

func (s *StdServer) Setup(ctx context.Context) error {
	s.config.Logger.Info(StdServerMod, "setting up web server...")

	dc := make(chan []error)
	s.config.EventManager.Publish(&ServerSetupEvent{S: s}, dc)

	errs := <-dc
	if len(errs) > 0 {
		errs = append(errs, fmt.Errorf("failed to setup server during event execution"))
		return errors.Join(errs...)
	}

	return nil
}

func (s *StdServer) Serve(ctx context.Context) error {
	s.config.Logger.Info(StdServerMod, "starting web server...")

	dc := make(chan []error)
	s.config.EventManager.Publish(&ServerStartEvent{S: s}, dc)

	errs := <-dc
	if len(errs) > 0 {
		errs = append(errs, fmt.Errorf("failed to start server during event execution"))
		return errors.Join(errs...)
	}

	return http.ListenAndServe(s.config.Addr, s.config.Router)
}

func (s *StdServer) RegisterController(c ...Controller) {
	s.config.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello World!")
		if err != nil {
			s.config.Logger.Error(StdServerMod, "failed to write response", "error", err)
		}
	})
}

func (s *StdServer) RegisterMiddleware(m ...func(http.Handler) http.Handler) {
	s.config.Router.Use(m...)
}
